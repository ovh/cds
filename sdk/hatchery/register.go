package hatchery

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/patrickmn/go-cache"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Create creates hatchery
func Create(h Interface, name, api, token string, maxWorkers int64, provisionDisabled bool, requestSecondsTimeout int, maxFailures int, insecureSkipVerifyTLS bool, provisionSeconds, registerSeconds, warningSeconds, criticalSeconds, graceSeconds int) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Gracefully shutdown connections
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer func() {
		signal.Stop(c)
		cancel()
	}()

	go func() {
		select {
		case <-c:
			defer cancel()
			return
		case <-ctx.Done():
			return
		}
	}()

	if err := h.Init(name, api, token, registerSeconds, insecureSkipVerifyTLS); err != nil {
		log.Error("Create> Init error: %s", err)
		os.Exit(10)
	}

	hostname, errh := os.Hostname()
	if errh != nil {
		log.Error("Create> Cannot retrieve hostname: %s", errh)
		os.Exit(10)
	}

	go hearbeat(h, token, maxFailures)

	pbjobs := make(chan sdk.PipelineBuildJob, 1)
	wjobs := make(chan sdk.WorkflowNodeJobRun, 1)
	errs := make(chan error, 1)
	var nRoutines, workersStarted int64

	go func(ctx context.Context) {
		if err := h.Client().QueuePolling(ctx, wjobs, pbjobs, errs, 2*time.Second); err != nil {
			log.Error("Queues polling stopped: %v", err)
		}
	}(ctx)

	// Create a cache with a default expiration time of 3 second, and which
	// purges expired items every minute
	spawnIDs := cache.New(3*time.Second, 60*time.Second)

	tickerProvision := time.NewTicker(time.Duration(provisionSeconds) * time.Second)
	tickerRegister := time.NewTicker(time.Duration(registerSeconds) * time.Second)
	tickerCountWorkersStarted := time.NewTicker(time.Duration(2 * time.Second))
	tickerGetModels := time.NewTicker(time.Duration(3 * time.Second))

	var maxWorkersReached bool
	var models []sdk.Model

	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Error("Exiting Hatchery: %v", err)
			} else {
				log.Info("Exiting Hatchery")
			}
			tickerRegister.Stop()
			return
		case <-tickerCountWorkersStarted.C:
			workersStarted = int64(h.WorkersStarted())
			if workersStarted > maxWorkers {
				log.Info("max workers reached. current:%d max:%d", workersStarted, maxWorkers)
				maxWorkersReached = true
			} else {
				maxWorkersReached = false
			}
			log.Debug("workers already started:%d", workersStarted)
		case <-tickerGetModels.C:
			var errwm error
			models, errwm = h.Client().WorkerModelsEnabled()
			if errwm != nil {
				log.Error("error on h.Client().WorkerModelsEnabled():%e", errwm)
			}
		case j := <-pbjobs:
			if maxWorkersReached {
				log.Debug("maxWorkerReached:%d", workersStarted)
				continue
			}
			go func(job sdk.PipelineBuildJob) {
				if isRun := receiveJob(h, job.ID, job.QueuedSeconds, job.BookedBy, job.Job.Action.Requirements, models, &nRoutines, spawnIDs, warningSeconds, criticalSeconds, graceSeconds, hostname); isRun {
					atomic.AddInt64(&workersStarted, 1)
					spawnIDs.SetDefault(string(job.ID), job.ID)
				}
			}(j)
		case j := <-wjobs:
			if maxWorkersReached {
				log.Debug("maxWorkerReached:%d", workersStarted)
				continue
			}
			go func(job sdk.WorkflowNodeJobRun) {
				if isRun := receiveJob(h, job.ID, job.QueuedSeconds, job.BookedBy, job.Job.Action.Requirements, models, &nRoutines, spawnIDs, warningSeconds, criticalSeconds, graceSeconds, hostname); isRun {
					atomic.AddInt64(&workersStarted, 1)
					spawnIDs.SetDefault(string(job.ID), job.ID)
				}
			}(j)
		case err := <-errs:
			log.Error("%v", err)
		case <-tickerProvision.C:
			provisioning(h, provisionDisabled, models)
		case <-tickerRegister.C:
			if err := workerRegister(h, models); err != nil {
				log.Warning("Error on workerRegister: %s", err)
			}
		}
	}
}

// Register calls CDS API to register current hatchery
func Register(h Interface) error {
	newHatchery, uptodate, err := h.Client().HatcheryRegister(*h.Hatchery())
	if err != nil {
		return sdk.WrapError(err, "register> Got HTTP exiting")
	}
	h.Hatchery().ID = newHatchery.ID
	h.Hatchery().GroupID = newHatchery.GroupID
	h.Hatchery().Model = newHatchery.Model
	sdk.Authorization(newHatchery.UID)
	sdk.SetAgent(sdk.HatcheryAgent)
	log.Info("Register> Hatchery %s registered with id:%d", h.Hatchery().Name, h.Hatchery().ID)

	if !uptodate {
		log.Warning("-=-=-=-=- Please update your hatchery binary -=-=-=-=-")
	}
	return nil
}

// GenerateName generate a hatchery's name
func GenerateName(add, name string) string {
	if name == "" {
		var errHostname error
		name, errHostname = os.Hostname()
		if errHostname != nil {
			log.Warning("Cannot retrieve hostname: %s", errHostname)
			name = "cds-hatchery"
		}
		name += "-" + namesgenerator.GetRandomName(0)
	}

	if add != "" {
		name += "-" + add
	}

	return name
}

func hearbeat(m Interface, token string, maxFailures int) {
	var failures int
	for {
		time.Sleep(5 * time.Second)
		if m.Hatchery().ID == 0 {
			log.Info("hearbeat> %s Disconnected from CDS engine, trying to register...", m.Hatchery().Name)
			if err := Register(m); err != nil {
				log.Info("hearbeat> %s Cannot register: %s", m.Hatchery().Name, err)
				checkFailures(maxFailures, failures)
				continue
			}
			if m.Hatchery().ID == 0 {
				log.Error("hearbeat> Cannot register hatchery. ID %d", m.Hatchery().ID)
				checkFailures(maxFailures, failures)
				continue
			}
			log.Info("hearbeat> %s Registered back: ID %d with model ID %d", m.Hatchery().Name, m.Hatchery().ID, m.Hatchery().Model.ID)
		}

		if _, _, err := sdk.Request("PUT", fmt.Sprintf("/hatchery/%d", m.Hatchery().ID), nil); err != nil {
			log.Info("heartbeat> %s cannot refresh beat: %s", m.Hatchery().Name, err)
			m.Hatchery().ID = 0
			checkFailures(maxFailures, failures)
			continue
		}
		failures = 0
	}
}

func checkFailures(maxFailures, nb int) {
	if nb > maxFailures {
		log.Error("Too many failures on try register. This hatchery is killed")
		os.Exit(10)
	}
}

func workerRegister(h Interface, models []sdk.Model) error {
	if len(models) == 0 {
		return fmt.Errorf("workerRegister> No model returned by GetWorkerModels")
	}
	log.Debug("workerRegister> models received: %d", len(models))

	var nRegistered int
	for _, m := range models {
		if m.Type != h.ModelType() {
			continue
		}

		// limit to 5 registration per ticker
		if nRegistered > 5 {
			break
		}
		// if current hatchery is in same group than worker model -> do not avoid spawn, even if worker model is in error
		if m.NbSpawnErr > 5 && h.Hatchery().GroupID != m.ID {
			log.Warning("workerRegister> Too many errors on spawn with model %s, please check this worker model", m.Name)
			continue
		}
		if h.NeedRegistration(&m) {
			log.Info("workerRegister> spawn a worker for register worker model %s (%d)", m.Name, m.ID)
			if _, errSpawn := h.SpawnWorker(&m, 0, nil, true, "spawn for register"); errSpawn != nil {
				log.Warning("workerRegister> cannot spawn worker for register: %s", m.Name, errSpawn)
				if err := sdk.SpawnErrorWorkerModel(m.ID, fmt.Sprintf("workerRegister> cannot spawn worker for register: %s", errSpawn)); err != nil {
					log.Error("workerRegister> error on call sdk.SpawnErrorWorkerModel on worker model %s for register: %s", m.Name, errSpawn)
				}
				continue
			}
			nRegistered++
		} else {
			log.Debug("workerRegister> no need to register worker model %s (%d)", m.Name, m.ID)
			continue
		}
	}
	return nil
}
