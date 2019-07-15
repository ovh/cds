package hatchery

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/namesgenerator"
)

type workerStarterRequest struct {
	ctx                 context.Context
	cancel              func(reason string)
	id                  int64
	model               *sdk.Model
	execGroups          []sdk.Group
	requirements        []sdk.Requirement
	hostname            string
	timestamp           int64
	workflowNodeRunID   int64
	registerWorkerModel *sdk.Model
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
func startWorkerStarters(ctx context.Context, h Interface) chan<- workerStarterRequest {
	jobs := make(chan workerStarterRequest, 1)

	maxProv := h.Configuration().Provision.MaxConcurrentProvisioning
	if maxProv < 1 {
		maxProv = defaultMaxProvisioning
	}
	for workerNum := 0; workerNum < maxProv; workerNum++ {
		sdk.GoRoutine(ctx, "workerStarter", func(ctx context.Context) {
			workerStarter(ctx, h, fmt.Sprintf("%d", workerNum), jobs)
		}, PanicDump(h))
	}
	return jobs
}

func workerStarter(ctx context.Context, h Interface, workerNum string, jobs <-chan workerStarterRequest) {
	for j := range jobs {
		// Start a worker for a job
		if m := j.registerWorkerModel; m == nil {
			_ = spawnWorkerForJob(h, j)
			j.cancel("")
		} else { // Start a worker for registering
			log.Debug("Spawning worker for register model %s", m.Name)
			if atomic.LoadInt64(&nbWorkerToStart) > int64(h.Configuration().Provision.MaxConcurrentProvisioning) {
				continue
			}

			atomic.AddInt64(&nbWorkerToStart, 1)
			// increment nbRegisteringWorkerModels, but no decrement.
			// this counter is reset with func workerRegister
			atomic.AddInt64(&nbRegisteringWorkerModels, 1)
			arg := SpawnArguments{
				WorkerName:   fmt.Sprintf("register-%s-%s", strings.ToLower(m.Name), strings.Replace(namesgenerator.GetRandomNameCDS(0), "_", "-", -1)),
				Model:        m,
				RegisterOnly: true,
				HatcheryName: h.ServiceName(),
			}

			// Get a JWT to authentified the worker
			_, jwt, err := NewWorkerToken(h.ServiceName(), h.PrivateKey(), time.Now().Add(1*time.Hour), arg)
			if err != nil {
				var spawnError = sdk.SpawnErrorForm{
					Error: fmt.Sprintf("cannot spawn worker for register: %v", err),
				}
				if err := h.CDSClient().WorkerModelSpawnError(m.Group.Name, m.Name, spawnError); err != nil {
					log.Error("workerStarter> error on call client.WorkerModelSpawnError on worker model %s for register: %s", m.Name, err)
				}
				continue
			}
			arg.WorkerToken = jwt

			if err := h.SpawnWorker(j.ctx, arg); err != nil {
				log.Warning("workerRegister> cannot spawn worker for register:%s err:%v", m.Name, err)
				var spawnError = sdk.SpawnErrorForm{
					Error: fmt.Sprintf("cannot spawn worker for register: %v", err),
				}
				if err := h.CDSClient().WorkerModelSpawnError(m.Group.Name, m.Name, spawnError); err != nil {
					log.Error("workerRegister> error on call client.WorkerModelSpawnError on worker model %s for register: %s", m.Name, err)
				}
			}
			atomic.AddInt64(&nbWorkerToStart, -1)
		}
	}
}

func spawnWorkerForJob(h Interface, j workerStarterRequest) bool {
	ctx, end := observability.Span(j.ctx, "hatchery.spawnWorkerForJob")
	defer end()

	stats.Record(WithTags(ctx, h), h.Metrics().SpawnedWorkers.M(1))

	log.Debug("hatchery> spawnWorkerForJob> %d", j.id)
	defer log.Debug("hatchery> spawnWorkerForJob> %d (%.3f seconds elapsed)", j.id, time.Since(time.Unix(j.timestamp, 0)).Seconds())

	maxProv := h.Configuration().Provision.MaxConcurrentProvisioning
	if maxProv < 1 {
		maxProv = defaultMaxProvisioning
	}
	if atomic.LoadInt64(&nbWorkerToStart) >= int64(maxProv) {
		log.Debug("hatchery> spawnWorkerForJob> max concurrent provisioning reached")
		return false
	}

	atomic.AddInt64(&nbWorkerToStart, 1)
	defer func(i *int64) {
		atomic.AddInt64(i, -1)
	}(&nbWorkerToStart)

	var modelName = "local"
	if j.model != nil {
		modelName = j.model.Group.Name + "+" + j.model.Name
	}

	if h.Service() == nil {
		log.Warning("hatchery> spawnWorkerForJob> %d - job %d %s- hatchery not registered", j.timestamp, j.id, modelName)
		return false
	}

	ctxQueueJobBook, next := observability.Span(ctx, "hatchery.QueueJobBook")
	ctxQueueJobBook, cancel := context.WithTimeout(ctxQueueJobBook, 10*time.Second)
	if err := h.CDSClient().QueueJobBook(ctxQueueJobBook, j.id); err != nil {
		next()
		// perhaps already booked by another hatchery
		log.Info("hatchery> spawnWorkerForJob> %d - cannot book job %d: %s", j.timestamp, j.id, err)
		cancel()
		return false
	}
	next()
	cancel()
	log.Debug("hatchery> spawnWorkerForJob> %d - send book job %d", j.timestamp, j.id)

	ctxSendSpawnInfo, next := observability.Span(ctx, "hatchery.SendSpawnInfo", observability.Tag("msg", sdk.MsgSpawnInfoHatcheryStarts.ID))
	start := time.Now()
	SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
		ID: sdk.MsgSpawnInfoHatcheryStarts.ID,
		Args: []interface{}{
			h.Service().Name,
			modelName,
		},
	})
	next()

	log.Info("hatchery> spawnWorkerForJob> SpawnWorker> starting model %s for job %d", modelName, j.id)

	_, next = observability.Span(ctx, "hatchery.SpawnWorker")
	arg := SpawnArguments{
		WorkerName:   fmt.Sprintf("%s-%s", strings.ToLower(modelName), strings.Replace(namesgenerator.GetRandomNameCDS(0), "_", "-", -1)),
		Model:        j.model,
		JobID:        j.id,
		Requirements: j.requirements,
		HatcheryName: h.ServiceName(),
	}

	// Get a JWT to authentified the worker
	_, jwt, err := NewWorkerToken(h.ServiceName(), h.PrivateKey(), time.Now().Add(1*time.Hour), arg)
	if err != nil {
		var spawnError = sdk.SpawnErrorForm{
			Error: fmt.Sprintf("cannot spawn worker for register: %v", err),
		}
		if err := h.CDSClient().WorkerModelSpawnError(j.model.Group.Name, j.model.Name, spawnError); err != nil {
			log.Error("spawnWorkerForJob> error on call client.WorkerModelSpawnError on worker model %s for register: %s", j.model.Name, err)
		}
		return false
	}
	arg.WorkerToken = jwt

	errSpawn := h.SpawnWorker(j.ctx, arg)
	next()
	if errSpawn != nil {
		ctxSendSpawnInfo, next = observability.Span(ctx, "hatchery.QueueJobSendSpawnInfo", observability.Tag("status", "errSpawn"), observability.Tag("msg", sdk.MsgSpawnInfoHatcheryErrorSpawn.ID))
		SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
			ID:   sdk.MsgSpawnInfoHatcheryErrorSpawn.ID,
			Args: []interface{}{h.Service().Name, modelName, sdk.Round(time.Since(start), time.Second).String(), errSpawn.Error()},
		})
		log.Error("hatchery %s cannot spawn worker %s for job %d: %v", h.Service().Name, modelName, j.id, errSpawn)
		next()
		return false
	}

	ctxSendSpawnInfo, next = observability.Span(ctx, "hatchery.SendSpawnInfo", observability.Tag("msg", sdk.MsgSpawnInfoHatcheryStartsSuccessfully.ID))
	SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
		ID: sdk.MsgSpawnInfoHatcheryStartsSuccessfully.ID,
		Args: []interface{}{
			h.Service().Name,
			arg.WorkerName,
			sdk.Round(time.Since(start), time.Second).String()},
	})
	next()

	if j.model != nil && j.model.IsDeprecated {
		ctxSendSpawnInfo, next = observability.Span(ctx, "hatchery.SendSpawnInfo", observability.Tag("msg", sdk.MsgSpawnInfoDeprecatedModel.ID))
		SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
			ID:   sdk.MsgSpawnInfoDeprecatedModel.ID,
			Args: []interface{}{modelName},
		})
		next()
	}
	return true // ok for this job
}
