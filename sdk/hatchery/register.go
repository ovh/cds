package hatchery

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Create creates hatchery
func Create(h Interface) {
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

	if err := h.Init(); err != nil {
		log.Error("Create> Init error: %s", err)
		os.Exit(10)
	}

	hostname, errh := os.Hostname()
	if errh != nil {
		log.Error("Create> Cannot retrieve hostname: %s", errh)
		os.Exit(10)
	}

	go hearbeat(h, h.Configuration().API.Token, h.Configuration().API.MaxHeartbeatFailures)

	pbjobs := make(chan sdk.PipelineBuildJob, 1)
	wjobs := make(chan sdk.WorkflowNodeJobRun, 1)
	errs := make(chan error, 1)
	var nRoutines, workersStarted int64

	go func(ctx context.Context) {
		if err := h.Client().QueuePolling(ctx, wjobs, pbjobs, errs, 2*time.Second, h.Configuration().Provision.GraceTimeQueued); err != nil {
			log.Error("Queues polling stopped: %v", err)
		}
	}(ctx)

	// Create a cache with a default expiration time of 3 second, and which
	// purges expired items every minute
	spawnIDs := cache.New(3*time.Second, 60*time.Second)

	tickerProvision := time.NewTicker(time.Duration(h.Configuration().Provision.Frequency) * time.Second)
	tickerRegister := time.NewTicker(time.Duration(h.Configuration().Provision.RegisterFrequency) * time.Second)
	tickerCountWorkersStarted := time.NewTicker(time.Duration(2 * time.Second))
	tickerGetModels := time.NewTicker(time.Duration(3 * time.Second))

	var models []sdk.Model

	// Call WorkerModel Enabled first
	var errwm error
	models, errwm = h.Client().WorkerModelsEnabled()
	if errwm != nil {
		log.Error("error on h.Client().WorkerModelsEnabled() (init call): %v", errwm)
	}

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
			if workersStarted > int64(h.Configuration().Provision.MaxWorker) {
				log.Info("max workers reached. current:%d max:%d", workersStarted, int64(h.Configuration().Provision.MaxWorker))
			}
			log.Debug("workers already started:%d", workersStarted)
		case <-tickerGetModels.C:
			var errwm error
			models, errwm = h.Client().WorkerModelsEnabled()
			if errwm != nil {
				log.Error("error on h.Client().WorkerModelsEnabled(): %v", errwm)
			}
		case j := <-pbjobs:
			if workersStarted > int64(h.Configuration().Provision.MaxWorker) {
				log.Debug("maxWorkersReached:%d", workersStarted)
				continue
			}
			go func(job sdk.PipelineBuildJob) {
				atomic.AddInt64(&workersStarted, 1)
				if isRun := receiveJob(h, false, job.ExecGroups, job.ID, job.QueuedSeconds, job.BookedBy, job.Job.Action.Requirements, models, &nRoutines, spawnIDs, hostname); isRun {
					spawnIDs.SetDefault(string(job.ID), job.ID)
				} else {
					atomic.AddInt64(&workersStarted, -1)
				}
			}(j)
		case j := <-wjobs:
			if workersStarted > int64(h.Configuration().Provision.MaxWorker) {
				log.Debug("maxWorkersReached:%d", workersStarted)
				continue
			}
			go func(job sdk.WorkflowNodeJobRun) {
				// count + 1 here, and remove -1 if worker is not started
				// this avoid to spawn to many workers compare
				atomic.AddInt64(&workersStarted, 1)
				if isRun := receiveJob(h, true, nil, job.ID, job.QueuedSeconds, job.BookedBy, job.Job.Action.Requirements, models, &nRoutines, spawnIDs, hostname); isRun {
					atomic.AddInt64(&workersStarted, 1)
					spawnIDs.SetDefault(string(job.ID), job.ID)
				} else {
					atomic.AddInt64(&workersStarted, -1)
				}
			}(j)
		case err := <-errs:
			log.Error("%v", err)
		case <-tickerProvision.C:
			provisioning(h, h.Configuration().Provision.Disabled, models)
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
	h.Hatchery().Name = newHatchery.Name
	h.Hatchery().IsSharedInfra = newHatchery.IsSharedInfra

	log.Info("Register> Hatchery %s registered with id:%d", h.Hatchery().Name, h.Hatchery().ID)

	if !uptodate {
		log.Warning("-=-=-=-=- Please update your hatchery binary - Hatchery Version:%s -=-=-=-=-", sdk.VERSION)
	}
	return nil
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

		if err := m.Client().HatcheryRefresh(m.Hatchery().ID); err != nil {
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
			if _, errSpawn := h.SpawnWorker(SpawnArguments{Model: m, IsWorkflowJob: false, JobID: 0, Requirements: nil, RegisterOnly: true, LogInfo: "spawn for register"}); errSpawn != nil {
				log.Warning("workerRegister> cannot spawn worker for register: %s", m.Name, errSpawn)
				if err := h.Client().WorkerModelSpawnError(m.ID, fmt.Sprintf("workerRegister> cannot spawn worker for register: %s", errSpawn)); err != nil {
					log.Error("workerRegister> error on call client.WorkerModelSpawnError on worker model %s for register: %s", m.Name, errSpawn)
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
