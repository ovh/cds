package hatchery

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/namesgenerator"
	"github.com/ovh/cds/sdk/slug"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
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

// Start all goroutines which manage the hatchery worker spawning routine.
// the purpose is to avoid go routines leak when there is a bunch of worker to start
func startWorkerStarters(ctx context.Context, h Interface) chan<- workerStarterRequest {
	jobs := make(chan workerStarterRequest, 1)

	maxProv := h.Configuration().Provision.MaxConcurrentProvisioning
	if maxProv < 1 {
		maxProv = defaultMaxProvisioning
	}
	for workerNum := 0; workerNum < maxProv; workerNum++ {
		workerNumStr := fmt.Sprintf("%d", workerNum)
		h.GetGoRoutines().Run(ctx, "workerStarter-"+workerNumStr, func(ctx context.Context) {
			workerStarter(ctx, h, workerNumStr, jobs)
		})
	}
	return jobs
}

func workerStarter(ctx context.Context, h Interface, workerNum string, jobs <-chan workerStarterRequest) {
	for j := range jobs {
		// Start a worker for a job
		if m := j.registerWorkerModel; m == nil {
			_ = spawnWorkerForJob(ctx, h, j)
			j.cancel("") // call to EndTrace for observability
		} else { // Start a worker for registering
			log.Debug(ctx, "Spawning worker for register model %s", m.Name)
			if atomic.LoadInt64(&nbWorkerToStart) > int64(h.Configuration().Provision.MaxConcurrentProvisioning) {
				continue
			}

			atomic.AddInt64(&nbWorkerToStart, 1)
			// increment nbRegisteringWorkerModels, but no decrement.
			// this counter is reset with func workerRegister
			atomic.AddInt64(&nbRegisteringWorkerModels, 1)
			arg := SpawnArguments{
				WorkerName:   generateWorkerName(h.Service().Name, true, m.Name),
				Model:        m,
				RegisterOnly: true,
				HatcheryName: h.Service().Name,
			}

			// Get a JWT to authentified the worker
			jwt, err := NewWorkerToken(h.Service().Name, h.GetPrivateKey(), time.Now().Add(1*time.Hour), arg)
			if err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				var spawnError = sdk.SpawnErrorForm{
					Error: fmt.Sprintf("cannot spawn worker for register: %v", err),
				}
				if err := h.CDSClient().WorkerModelSpawnError(m.Group.Name, m.Name, spawnError); err != nil {
					log.Error(ctx, "workerStarter> error on call client.WorkerModelSpawnError on worker model %s for register: %s", m.Name, err)
				}
				continue
			}
			arg.WorkerToken = jwt

			if err := h.SpawnWorker(ctx, arg); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Warn(ctx, "workerRegister> cannot spawn worker for register:%s err:%v", m.Name, err)
				var spawnError = sdk.SpawnErrorForm{
					Error: fmt.Sprintf("cannot spawn worker for register: %v", err),
				}
				if err := h.CDSClient().WorkerModelSpawnError(m.Group.Name, m.Name, spawnError); err != nil {
					log.Error(ctx, "workerRegister> error on call client.WorkerModelSpawnError on worker model %s for register: %s", m.Name, err)
				}
			}
			atomic.AddInt64(&nbWorkerToStart, -1)
		}
	}
}

func spawnWorkerForJob(ctx context.Context, h Interface, j workerStarterRequest) bool {
	ctxJob, end := telemetry.Span(j.ctx, "hatchery.spawnWorkerForJob")
	defer end()

	ctxJob = telemetry.ContextWithTag(ctxJob,
		telemetry.TagServiceName, h.Name(),
		telemetry.TagServiceType, h.Type(),
	)
	telemetry.Record(ctxJob, GetMetrics().SpawnedWorkers, 1)

	log.Debug(ctx, "hatchery> spawnWorkerForJob> %d", j.id)
	defer log.Debug(ctx, "hatchery> spawnWorkerForJob> %d (%.3f seconds elapsed)", j.id, time.Since(time.Unix(j.timestamp, 0)).Seconds())

	maxProv := h.Configuration().Provision.MaxConcurrentProvisioning
	if maxProv < 1 {
		maxProv = defaultMaxProvisioning
	}
	if atomic.LoadInt64(&nbWorkerToStart) >= int64(maxProv) {
		log.Debug(ctx, "hatchery> spawnWorkerForJob> max concurrent provisioning reached")
		return false
	}

	atomic.AddInt64(&nbWorkerToStart, 1)
	defer func(i *int64) {
		atomic.AddInt64(i, -1)
	}(&nbWorkerToStart)

	var modelName = "local"
	if j.model != nil {
		modelName = j.model.Group.Name + "/" + j.model.Name
	}

	if h.Service() == nil {
		log.Warn(ctx, "hatchery> spawnWorkerForJob> %d - job %d %s- hatchery not registered", j.timestamp, j.id, modelName)
		return false
	}

	ctxQueueJobBook, next := telemetry.Span(ctxJob, "hatchery.QueueJobBook")
	ctxQueueJobBook, cancel := context.WithTimeout(ctxQueueJobBook, 10*time.Second)
	bookedInfos, err := h.CDSClient().QueueJobBook(ctxQueueJobBook, j.id)
	if err != nil {
		next()
		// perhaps already booked by another hatchery
		log.Info(ctx, "hatchery> spawnWorkerForJob> %d - cannot book job %d: %s", j.timestamp, j.id, err)
		cancel()
		return false
	}
	next()
	cancel()
	log.Debug(ctx, "hatchery> spawnWorkerForJob> %d - send book job %d", j.timestamp, j.id)

	ctxSendSpawnInfo, next := telemetry.Span(ctxJob, "hatchery.SendSpawnInfo", telemetry.Tag("msg", sdk.MsgSpawnInfoHatcheryStarts.ID))
	start := time.Now()
	SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
		ID: sdk.MsgSpawnInfoHatcheryStarts.ID,
		Args: []interface{}{
			h.Service().Name,
			modelName,
		},
	})
	next()

	_, next = telemetry.Span(ctxJob, "hatchery.SpawnWorker")
	arg := SpawnArguments{
		WorkerName:   generateWorkerName(h.Service().Name, false, modelName),
		Model:        j.model,
		JobID:        j.id,
		NodeRunID:    j.workflowNodeRunID,
		Requirements: j.requirements,
		HatcheryName: h.Service().Name,
		NodeRunName:  bookedInfos.NodeRunName,
		RunID:        bookedInfos.RunID,
		WorkflowID:   bookedInfos.WorkflowID,
		WorkflowName: bookedInfos.WorkflowName,
		ProjectKey:   bookedInfos.ProjectKey,
		JobName:      bookedInfos.JobName,
	}

	log.Info(ctx, "hatchery> spawnWorkerForJob> SpawnWorker> starting model %s for job %d with name %s", modelName, arg.JobID, arg.WorkerName)

	// Get a JWT to authentified the worker
	jwt, err := NewWorkerToken(h.Service().Name, h.GetPrivateKey(), time.Now().Add(1*time.Hour), arg)
	if err != nil {
		ctx = sdk.ContextWithStacktrace(ctx, err)
		var spawnError = sdk.SpawnErrorForm{
			Error: fmt.Sprintf("cannot spawn worker for register: %v", err),
		}
		if err := h.CDSClient().WorkerModelSpawnError(j.model.Group.Name, j.model.Name, spawnError); err != nil {
			log.Error(ctx, "hatchery> spawnWorkerForJob> error on call client.WorkerModelSpawnError on worker model %s for register: %s", j.model.Name, err)
		}
		return false
	}
	arg.WorkerToken = jwt
	log.Debug(ctx, "hatchery> spawnWorkerForJob> new JWT for worker: %s", jwt)

	errSpawn := h.SpawnWorker(ctx, arg)
	next()
	if errSpawn != nil {
		ctx = sdk.ContextWithStacktrace(ctx, errSpawn)
		ctxSendSpawnInfo, next = telemetry.Span(ctxJob, "hatchery.QueueJobSendSpawnInfo", telemetry.Tag("status", "errSpawn"), telemetry.Tag("msg", sdk.MsgSpawnInfoHatcheryErrorSpawn.ID))
		SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
			ID:   sdk.MsgSpawnInfoHatcheryErrorSpawn.ID,
			Args: []interface{}{h.Service().Name, modelName, sdk.Round(time.Since(start), time.Second).String(), sdk.ExtractHTTPError(errSpawn).Error()},
		})
		log.Error(ctx, "hatchery %s cannot spawn worker %s for job %d: %v", h.Service().Name, modelName, j.id, errSpawn)
		next()
		return false
	}

	ctxSendSpawnInfo, next = telemetry.Span(ctxJob, "hatchery.SendSpawnInfo", telemetry.Tag("msg", sdk.MsgSpawnInfoHatcheryStartsSuccessfully.ID))
	SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
		ID: sdk.MsgSpawnInfoHatcheryStartsSuccessfully.ID,
		Args: []interface{}{
			h.Service().Name,
			arg.WorkerName,
			sdk.Round(time.Since(start), time.Second).String()},
	})
	next()

	if j.model != nil && j.model.IsDeprecated {
		ctxSendSpawnInfo, next = telemetry.Span(ctxJob, "hatchery.SendSpawnInfo", telemetry.Tag("msg", sdk.MsgSpawnInfoDeprecatedModel.ID))
		SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
			ID:   sdk.MsgSpawnInfoDeprecatedModel.ID,
			Args: []interface{}{modelName},
		})
		next()
	}
	return true // ok for this job
}

// a worker name must be 60 char max, without '.' and '_', "/" -> replaced by '-'
func generateWorkerName(hatcheryName string, isRegister bool, modelName string) string {
	prefix := ""
	if isRegister {
		prefix = "register-"
	}

	maxLength := 63
	hName := hatcheryName + "-"
	random := namesgenerator.GetRandomNameCDS(0)
	workerName := fmt.Sprintf("%s%s-%s-%s", prefix, hatcheryName, modelName, random)

	if len(workerName) <= maxLength {
		return slug.Convert(workerName)
	}
	if len(hName) > 10 {
		hName = ""
	}
	workerName = fmt.Sprintf("%s%s%s-%s", prefix, hName, modelName, random)
	if len(workerName) <= maxLength {
		return slug.Convert(workerName)
	}
	modelName = sdk.StringFirstN(modelName, 15)
	workerName = fmt.Sprintf("%s%s%s-%s", prefix, hName, modelName, random)
	return slug.Convert(sdk.StringFirstN(workerName, maxLength))
}
