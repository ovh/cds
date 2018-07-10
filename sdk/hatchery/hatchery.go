package hatchery

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	cache "github.com/patrickmn/go-cache"
)

var (
	// Client is a CDS Client
	Client                 sdk.HTTPClient
	defaultMaxProvisioning = 10
	models                 []sdk.Model
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

	sdk.GoRoutine("heartbeat", func() {
		hearbeat(h, h.Configuration().API.Token, h.Configuration().API.MaxHeartbeatFailures)
	})

	pbjobs := make(chan sdk.PipelineBuildJob, 1)
	wjobs := make(chan sdk.WorkflowNodeJobRun, 10)
	errs := make(chan error, 1)

	sdk.GoRoutine("queuePolling", func() {
		if err := h.CDSClient().QueuePolling(ctx, wjobs, pbjobs, errs, 2*time.Second, h.Configuration().Provision.GraceTimeQueued, nil); err != nil {
			log.Error("Queues polling stopped: %v", err)
			cancel()
		}
	})

	// Create a cache with a default expiration time of 3 second, and which
	// purges expired items every minute
	spawnIDs := cache.New(10*time.Second, 60*time.Second)
	// This is a local cache to avoid analysing a job twice at the same time
	receivedIDs := cache.New(5*time.Second, 60*time.Second)

	tickerProvision := time.NewTicker(time.Duration(h.Configuration().Provision.Frequency) * time.Second)
	tickerRegister := time.NewTicker(time.Duration(h.Configuration().Provision.RegisterFrequency) * time.Second)
	tickerGetModels := time.NewTicker(time.Duration(3 * time.Second))

	defer func() {
		tickerProvision.Stop()
		tickerRegister.Stop()
		tickerGetModels.Stop()
	}()

	// Call WorkerModel Enabled first
	var errwm error
	models, errwm = h.CDSClient().WorkerModelsEnabled()
	if errwm != nil {
		log.Error("error on h.CDSClient().WorkerModelsEnabled() (init call): %v", errwm)
	}

	// hatchery is now fully Initialized
	h.SetInitialized()

	// run the starters pool
	workersStartChan, workerStartResultChan := startWorkerStarters(h)

	// read the result channel in another goroutine to let the main goroutine start new workers
	sdk.GoRoutine("checkStarterResult", func() {
		for startWorkerRes := range workerStartResultChan {
			receivedIDs.Delete(string(startWorkerRes.request.id))
			if startWorkerRes.err != nil {
				errs <- startWorkerRes.err
			}
			if startWorkerRes.isRun {
				spawnIDs.SetDefault(string(startWorkerRes.request.id), startWorkerRes.request.id)
			} else if startWorkerRes.temptToSpawn {
				found := false
				for _, hID := range startWorkerRes.request.spawnAttempts {
					if hID == h.ID() {
						found = true
					}
				}
				if !found {
					if hCount, err := h.CDSClient().HatcheryCount(startWorkerRes.request.workflowNodeRunID); err == nil {
						if int64(len(startWorkerRes.request.spawnAttempts)) < hCount {
							if _, errQ := h.CDSClient().QueueJobIncAttempts(startWorkerRes.request.id); errQ != nil {
								log.Warning("Hatchery> Create> cannot inc spawn attempts %v", errQ)
							}
						}
					} else {
						log.Warning("Hatchery> Create> cannot get hatchery count: %v", err)
					}
				}
			}
		}
	})

	// the main goroutine
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-tickerGetModels.C:

			var errwm error
			models, errwm = h.CDSClient().WorkerModelsEnabled()
			if errwm != nil {
				log.Error("error on h.CDSClient().WorkerModelsEnabled(): %v", errwm)
			}

		case j := <-pbjobs:
			if j.ID == 0 {
				continue
			}

			//Check bookedBy current hatchery
			if j.BookedBy.ID == 0 || j.BookedBy.ID != h.ID() {
				continue
			}

			//Check gracetime
			if j.QueuedSeconds < int64(h.Configuration().Provision.GraceTimeQueued) {
				log.Debug("job %d is too fresh, queued since %d seconds, let existing waiting worker check it", j.ID, j.QueuedSeconds)
				continue
			}

			//Check if hatchery if able to provision
			if !checkProvisioning(h) {
				log.Info("hatchery is not able to provision new worker")
				continue
			}

			//Check spawnsID
			if _, exist := spawnIDs.Get(string(j.ID)); exist {
				log.Debug("job %d already spawned in previous routine", j.ID)
				continue
			}

			//Ask to start
			workersStartChan <- workerStarterRequest{
				id:            j.ID,
				isWorkflowJob: false,
				execGroups:    j.ExecGroups,
				models:        models,
				requirements:  j.Job.Action.Requirements,
				hostname:      hostname,
				timestamp:     time.Now().Unix(),
			}

		case j := <-wjobs:
			if j.ID == 0 {
				continue
			}

			if _, exist := receivedIDs.Get(string(j.ID)); exist {
				log.Debug("job %d is alrealy being analyzed", j.ID)
				continue
			}
			receivedIDs.SetDefault(string(j.ID), j.ID)

			//Check bookedBy current hatchery
			if j.BookedBy.ID != 0 && j.BookedBy.ID != h.ID() {
				log.Debug("hatchery> job %d is booked by someone else (%d / %d)", j.ID, j.BookedBy.ID, h.ID())
				receivedIDs.Delete(string(j.ID))
				continue
			}

			//Check gracetime
			if j.QueuedSeconds < int64(h.Configuration().Provision.GraceTimeQueued) {
				log.Debug("job %d is too fresh, queued since %d seconds, let existing waiting worker check it", j.ID)
				receivedIDs.Delete(string(j.ID))
				continue
			}

			//Check spawnsID
			if _, exist := spawnIDs.Get(string(j.ID)); exist {
				log.Debug("job %d already spawned in previous routine", j.ID)
				receivedIDs.Delete(string(j.ID))
				continue
			}

			log.Debug("Analyzing job %d", j.ID)

			//Check if hatchery if able to provision
			if !checkProvisioning(h) {
				receivedIDs.Delete(string(j.ID))
				continue
			}

			//Ask to start
			log.Info("Request a worker for job %d", j.ID)

			workersStartChan <- workerStarterRequest{
				id:                j.ID,
				isWorkflowJob:     true,
				execGroups:        j.ExecGroups,
				models:            models,
				requirements:      j.Job.Action.Requirements,
				hostname:          hostname,
				timestamp:         time.Now().Unix(),
				spawnAttempts:     j.SpawnAttempts,
				workflowNodeRunID: j.WorkflowNodeRunID,
			}

		case err := <-errs:
			log.Error("%v", err)

		case <-tickerProvision.C:
			provisioning(h, h.Configuration().Provision.Disabled, models)

		case <-tickerRegister.C:
			if err := workerRegister(h, workersStartChan); err != nil {
				log.Warning("Error on workerRegister: %s", err)
			}
		}
	}
}

// MemoryRegisterContainer is the RAM used for spawning
// a docker container for register a worker model. 128 Mo
const MemoryRegisterContainer int64 = 128

// CheckRequirement checks binary requirement in path
func CheckRequirement(r sdk.Requirement) (bool, error) {
	switch r.Type {
	case sdk.BinaryRequirement:
		if _, err := exec.LookPath(r.Value); err != nil {
			// Return nil because the error contains 'Exit status X', that's what we wanted
			return false, nil
		}
		return true, nil
	default:
		return false, nil
	}
}

func logTime(h Interface, name string, then time.Time) {
	d := time.Since(then)
	if d > time.Duration(h.Configuration().LogOptions.SpawnOptions.ThresholdCritical)*time.Second {
		log.Error("%s took %s to execute", name, d)
		return
	}

	if d > time.Duration(h.Configuration().LogOptions.SpawnOptions.ThresholdWarning)*time.Second {
		log.Warning("%s took %s to execute", name, d)
		return
	}

	log.Debug("%s took %s to execute", name, d)
}
