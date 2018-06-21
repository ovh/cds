package hatchery

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Create creates hatchery
func Create(h Interface) error {
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
		return fmt.Errorf("Create> Init error: %v", err)
	}

	hostname, errh := os.Hostname()
	if errh != nil {
		return fmt.Errorf("Create> Cannot retrieve hostname: %s", errh)
	}

	go hearbeat(h, h.Configuration().API.Token, h.Configuration().API.MaxHeartbeatFailures)

	pbjobs := make(chan sdk.PipelineBuildJob, 1)
	wjobs := make(chan sdk.WorkflowNodeJobRun, 1)
	errs := make(chan error, 1)
	var nRoutines, workersStarted, nRegister int64

	go func(ctx context.Context) {
		if err := h.CDSClient().QueuePolling(ctx, wjobs, pbjobs, errs, 2*time.Second, h.Configuration().Provision.GraceTimeQueued, nil); err != nil {
			log.Error("Queues polling stopped: %v", err)
			cancel()
		}
	}(ctx)

	// Create a cache with a default expiration time of 3 second, and which
	// purges expired items every minute
	spawnIDs := cache.New(3*time.Second, 60*time.Second)

	tickerProvision := time.NewTicker(time.Duration(h.Configuration().Provision.Frequency) * time.Second)
	tickerRegister := time.NewTicker(time.Duration(h.Configuration().Provision.RegisterFrequency) * time.Second)
	tickerCountWorkersStarted := time.NewTicker(time.Duration(2 * time.Second))
	tickerGetModels := time.NewTicker(time.Duration(3 * time.Second))
	defer func() {
		tickerProvision.Stop()
		tickerRegister.Stop()
		tickerCountWorkersStarted.Stop()
		tickerGetModels.Stop()
	}()

	var models []sdk.Model

	// Call WorkerModel Enabled first
	var errwm error
	models, errwm = h.CDSClient().WorkerModelsEnabled()
	if errwm != nil {
		log.Error("error on h.CDSClient().WorkerModelsEnabled() (init call): %v", errwm)
	}

	// hatchery is now fully Initialized
	h.SetInitialized()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tickerCountWorkersStarted.C:
			workersStarted = int64(h.WorkersStarted())
			if workersStarted > int64(h.Configuration().Provision.MaxWorker) {
				log.Debug("max workers reached. current:%d max:%d", workersStarted, int64(h.Configuration().Provision.MaxWorker))
			}
			log.Debug("workers already started:%d", workersStarted)
		case <-tickerGetModels.C:
			var errwm error
			models, errwm = h.CDSClient().WorkerModelsEnabled()
			if errwm != nil {
				log.Error("error on h.CDSClient().WorkerModelsEnabled(): %v", errwm)
			}
		case j := <-pbjobs:
			if workersStarted > int64(h.Configuration().Provision.MaxWorker) {
				log.Debug("maxWorkersReached:%d", workersStarted)
				continue
			}
			go func(job sdk.PipelineBuildJob) {
				atomic.AddInt64(&workersStarted, 1)
				if isRun, _, _ := receiveJob(h, false, job.ExecGroups, job.ID, job.QueuedSeconds, []int64{}, job.BookedBy, job.Job.Action.Requirements, models, &nRoutines, spawnIDs, hostname); isRun {
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
				if isRun, temptToSpawn, _ := receiveJob(h, true, job.ExecGroups, job.ID, job.QueuedSeconds, job.SpawnAttempts, job.BookedBy, job.Job.Action.Requirements, models, &nRoutines, spawnIDs, hostname); isRun {
					atomic.AddInt64(&workersStarted, 1)
					spawnIDs.SetDefault(string(job.ID), job.ID)
				} else if temptToSpawn {
					atomic.AddInt64(&workersStarted, -1)
					found := false
					for _, hID := range job.SpawnAttempts {
						if hID == h.ID() {
							found = true
						}
					}

					if !found {
						if hCount, err := h.CDSClient().HatcheryCount(job.WorkflowNodeRunID); err == nil {
							if int64(len(job.SpawnAttempts)) < hCount {
								if _, errQ := h.CDSClient().QueueJobIncAttempts(job.ID); errQ != nil {
									log.Warning("Hatchery> Create> cannot inc spawn attempts %s", errQ)
								}
							}
						} else {
							log.Warning("Hatchery> Create> cannot get hatchery count %s", err)
						}
					}

				} else {
					atomic.AddInt64(&workersStarted, -1)
				}
			}(j)
		case err := <-errs:
			log.Error("%v", err)
		case <-tickerProvision.C:
			provisioning(h, h.Configuration().Provision.Disabled, models)
		case <-tickerRegister.C:
			if err := workerRegister(h, models, &nRegister); err != nil {
				log.Warning("Error on workerRegister: %s", err)
			}
		}
	}
}

// Register calls CDS API to register current hatchery
func Register(h Interface) error {
	newHatchery, uptodate, err := h.CDSClient().HatcheryRegister(*h.Hatchery())
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
		log.Warning("-=-=-=-=- Please update your hatchery binary - Hatchery Version:%s %s %s -=-=-=-=-", sdk.VERSION, runtime.GOOS, runtime.GOARCH)
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

		if err := m.CDSClient().HatcheryRefresh(m.Hatchery().ID); err != nil {
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

// workerRegister is called by a ticker.
// the hatchery checks each worker model, and if a worker model needs to
// be registered, the hatchery calls SpawnWorker().
// each ticker can trigger 5 worker models (maximum)
// and 5 worker models can be spawned in same time, in the case of a spawn takes longer
// than a tick.
func workerRegister(h Interface, models []sdk.Model, nRegister *int64) error {
	if len(models) == 0 {
		return fmt.Errorf("workerRegister> No model returned by GetWorkerModels")
	}
	log.Debug("workerRegister> models received: %d", len(models))

	// currentRegister contains the register spawned in this ticker
	var currentRegister int64

	for k := range models {
		if *nRegister > 5 || currentRegister > 5 {
			return nil
		}

		if models[k].Type != h.ModelType() {
			continue
		}

		// if current hatchery is in same group than worker model -> do not avoid spawn, even if worker model is in error
		if models[k].NbSpawnErr > 5 && h.Hatchery().GroupID != models[k].ID {
			log.Warning("workerRegister> Too many errors on spawn with model %s, please check this worker model", models[k].Name)
			continue
		}

		if h.NeedRegistration(&models[k]) {
			if err := h.CDSClient().WorkerModelBook(models[k].ID); err != nil {
				log.Debug("workerRegister> WorkerModelBook on model %s err: %s", models[k].Name, err)
			} else {
				log.Info("workerRegister> spawning model %s (%d)", models[k].Name, models[k].ID)
				atomic.AddInt64(nRegister, 1)
				currentRegister++
				go func(m sdk.Model) {
					if _, errSpawn := h.SpawnWorker(SpawnArguments{Model: m, IsWorkflowJob: false, JobID: 0, Requirements: nil, RegisterOnly: true, LogInfo: "spawn for register"}); errSpawn != nil {
						log.Warning("workerRegister> cannot spawn worker for register:%s err:%v", m.Name, errSpawn)
						if err := h.CDSClient().WorkerModelSpawnError(m.ID, fmt.Sprintf("workerRegister> cannot spawn worker for register: %s", errSpawn)); err != nil {
							log.Error("workerRegister> error on call client.WorkerModelSpawnError on worker model %s for register: %s", m.Name, err)
						}
					}
					atomic.AddInt64(nRegister, -1)
				}(models[k])
			}
		} else {
			log.Debug("workerRegister> no need to register worker model %s (%d)", models[k].Name, models[k].ID)
		}
	}
	return nil
}
