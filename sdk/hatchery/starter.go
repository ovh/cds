package hatchery

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type workerStarterRequest struct {
	ctx                 context.Context
	cancel              func(reason string)
	id                  int64
	isWorkflowJob       bool
	model               sdk.Model
	execGroups          []sdk.Group
	requirements        []sdk.Requirement
	hostname            string
	timestamp           int64
	spawnAttempts       []int64
	workflowNodeRunID   int64
	registerWorkerModel *sdk.Model
}

type workerStarterResult struct {
	request      workerStarterRequest
	isRun        bool
	temptToSpawn bool
	err          error
}

func PanicDump(h Interface) func(s string) (io.WriteCloser, error) {
	return func(s string) (io.WriteCloser, error) {
		dir, err := h.PanicDumpDirectory()
		if err != nil {
			return nil, err
		}
		return os.OpenFile(filepath.Join(dir, s), os.O_RDWR|os.O_CREATE, 0644)
	}
}

// Start all goroutines which manage the hatchery worker spawning routine.
// the purpose is to avoid go routines leak when there is a bunch of worker to start
func startWorkerStarters(ctx context.Context, h Interface) (chan<- workerStarterRequest, chan workerStarterResult) {
	jobs := make(chan workerStarterRequest, 1)
	results := make(chan workerStarterResult, 1)

	maxProv := h.Configuration().Provision.MaxConcurrentProvisioning
	if maxProv < 1 {
		maxProv = defaultMaxProvisioning
	}
	for workerNum := 0; workerNum < maxProv; workerNum++ {
		sdk.GoRoutine("workerStarter",
			func() {
				workerStarter(ctx, h, fmt.Sprintf("%d", workerNum), jobs, results)
			},
			PanicDump(h),
		)
	}
	return jobs, results
}

func workerStarter(ctx context.Context, h Interface, workerNum string, jobs <-chan workerStarterRequest, results chan<- workerStarterResult) {
	for j := range jobs {
		// Start a worker for a job
		if m := j.registerWorkerModel; m == nil {
			ctx2, end := observability.Span(j.ctx, "hatchery.workerStarter")
			isRun, err := spawnWorkerForJob(h, j)
			//Check the result
			res := workerStarterResult{
				request:      j,
				err:          err,
				isRun:        isRun,
				temptToSpawn: true,
			}

			_, cend := observability.Span(ctx2, "sendResult")
			//Send the result back
			results <- res
			cend()

			if err != nil {
				j.cancel(err.Error())
			} else {
				j.cancel("")
			}
			end()
		} else { // Start a worker for registering
			log.Debug("Spawning worker for register model %s", m.Name)
			if atomic.LoadInt64(&nbWorkerToStart) > int64(h.Configuration().Provision.MaxConcurrentProvisioning) {
				continue
			}

			atomic.AddInt64(&nbWorkerToStart, 1)
			atomic.AddInt64(&nbRegisteringWorkerModels, 1)
			if _, errSpawn := h.SpawnWorker(j.ctx, SpawnArguments{Model: *m, IsWorkflowJob: false, JobID: 0, Requirements: nil, RegisterOnly: true, LogInfo: "spawn for register"}); errSpawn != nil {
				log.Warning("workerRegister> cannot spawn worker for register:%s err:%v", m.Name, errSpawn)
				if err := h.CDSClient().WorkerModelSpawnError(m.ID, fmt.Sprintf("cannot spawn worker for register: %s", errSpawn)); err != nil {
					log.Error("workerRegister> error on call client.WorkerModelSpawnError on worker model %s for register: %s", m.Name, err)
				}
			}
			atomic.AddInt64(&nbWorkerToStart, -1)
			atomic.AddInt64(&nbRegisteringWorkerModels, -1)
		}
	}
}

func spawnWorkerForJob(h Interface, j workerStarterRequest) (bool, error) {
	ctx, end := observability.Span(j.ctx, "hatchery.spawnWorkerForJob")
	defer end()

	stats.Record(WithTags(ctx, h), h.Stats().SpawnedWorkers.M(1))

	log.Debug("hatchery> spawnWorkerForJob> %d", j.id)
	defer log.Debug("hatchery> spawnWorkerForJob> %d (%.3f seconds elapsed)", j.id, time.Since(time.Unix(j.timestamp, 0)).Seconds())

	maxProv := h.Configuration().Provision.MaxConcurrentProvisioning
	if maxProv < 1 {
		maxProv = defaultMaxProvisioning
	}
	if atomic.LoadInt64(&nbWorkerToStart) >= int64(maxProv) {
		log.Debug("hatchery> spawnWorkerForJob> max concurrent provisioning reached")
		return false, nil
	}

	atomic.AddInt64(&nbWorkerToStart, 1)
	defer func(i *int64) {
		atomic.AddInt64(i, -1)
	}(&nbWorkerToStart)

	if h.CDSClient().GetService() == nil || h.ID() == 0 {
		log.Warning("hatchery> spawnWorkerForJob> %d - job %d %s- hatchery not registered - srv:%t id:%d", j.timestamp, j.id, j.model.Name, h.CDSClient().GetService() == nil, h.ID())
		return false, nil
	}

	_, next := observability.Span(ctx, "hatchery.QueueJobBook")
	ctxt, cancel := context.WithTimeout(ctx, 10*time.Second)
	if err := h.CDSClient().QueueJobBook(ctxt, j.isWorkflowJob, j.id); err != nil {
		next()
		// perhaps already booked by another hatchery
		log.Info("hatchery> spawnWorkerForJob> %d - cannot book job %d %s: %s", j.timestamp, j.id, j.model.Name, err)
		cancel()
		return false, nil
	}
	next()
	cancel()
	log.Debug("hatchery> spawnWorkerForJob> %d - send book job %d %s by hatchery %d isWorkflowJob:%t", j.timestamp, j.id, j.model.Name, h.ID(), j.isWorkflowJob)

	start := time.Now()
	infos := []sdk.SpawnInfo{
		{
			RemoteTime: start,
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStarts.ID, Args: []interface{}{h.Service().Name, fmt.Sprintf("%d", h.ID()), j.model.Name}},
		},
	}
	log.Info("hatchery> spawnWorkerForJob> SpawnWorker> starting model %s for job %d", j.model.Name, j.id)
	_, next = observability.Span(ctx, "hatchery.SpawnWorker")
	workerName, errSpawn := h.SpawnWorker(j.ctx, SpawnArguments{Model: j.model, IsWorkflowJob: j.isWorkflowJob, JobID: j.id, Requirements: j.requirements, LogInfo: "spawn for job"})
	next()
	if errSpawn != nil {
		_, next = observability.Span(ctx, "hatchery.QueueJobSendSpawnInfo", observability.Tag("status", "errSpawn"))
		log.Warning("spawnWorkerForJob> %d - cannot spawn worker %s for job %d: %s", j.timestamp, j.model.Name, j.id, errSpawn)
		infos = append(infos, sdk.SpawnInfo{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryErrorSpawn.ID, Args: []interface{}{h.Service().Name, fmt.Sprintf("%d", h.ID()), j.model.Name, sdk.Round(time.Since(start), time.Second).String(), errSpawn.Error()}},
		})
		ctxt, cancel := context.WithTimeout(ctx, 10*time.Second)
		if err := h.CDSClient().QueueJobSendSpawnInfo(ctxt, j.isWorkflowJob, j.id, infos); err != nil {
			log.Warning("spawnWorkerForJob> %d - cannot client.QueueJobSendSpawnInfo for job (err spawn)%d: %s", j.timestamp, j.id, err)
		}
		log.Error("hatchery %s cannot spawn worker %s for job %d: %v", h.Service().Name, j.model.Name, j.id, errSpawn)
		next()
		cancel()
		return false, nil
	}

	infos = append(infos, sdk.SpawnInfo{
		RemoteTime: time.Now(),
		Message: sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStartsSuccessfully.ID,
			Args: []interface{}{
				h.Service().Name,
				fmt.Sprintf("%d", h.ID()),
				workerName,
				sdk.Round(time.Since(start), time.Second).String()},
		},
	})

	ctxtJobSendSpawnInfo, cancelJobSendSpawnInfo := context.WithTimeout(ctx, 10*time.Second)
	_, next = observability.Span(ctx, "hatchery.QueueJobSendSpawnInfo", observability.Tag("status", "spawnOK"))
	if err := h.CDSClient().QueueJobSendSpawnInfo(ctxtJobSendSpawnInfo, j.isWorkflowJob, j.id, infos); err != nil {
		next()
		log.Warning("spawnWorkerForJob> %d - cannot client.QueueJobSendSpawnInfo for job %d: %s", j.timestamp, j.id, err)
	}
	next()
	cancelJobSendSpawnInfo()
	return true, nil // ok for this job
}
