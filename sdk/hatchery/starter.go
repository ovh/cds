package hatchery

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/namesgenerator"
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
			_ = spawnWorkerForJob(j.ctx, h, j)
			j.cancel("") // call to EndTrace for observability
		} else { // Start a worker for registering
			log.Debug(ctx, "Spawning worker for register model %s", m.Name)
			if atomic.LoadInt64(&nbWorkerToStart) > int64(h.Configuration().Provision.MaxConcurrentProvisioning) {
				continue
			}

			workerName := namesgenerator.GenerateWorkerName(m.Name, "register")

			atomic.AddInt64(&nbWorkerToStart, 1)
			// increment nbRegisteringWorkerModels, but no decrement.
			// this counter is reset with func workerRegister
			atomic.AddInt64(&nbRegisteringWorkerModels, 1)
			arg := SpawnArguments{
				WorkerName:   workerName,
				Model:        m,
				RegisterOnly: true,
				HatcheryName: h.Service().Name,
			}

			ctx = context.WithValue(ctx, cdslog.AuthWorkerName, arg.WorkerName)
			log.Info(ctx, "starting worker %q from model %q (register:%v)", arg.WorkerName, m.Name, arg.RegisterOnly)

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
	ctx, end := telemetry.Span(ctx, "hatchery.spawnWorkerForJob")
	defer end()

	ctx = telemetry.ContextWithTag(ctx,
		telemetry.TagServiceName, h.Name(),
		telemetry.TagServiceType, h.Type(),
	)
	telemetry.Record(ctx, GetMetrics().SpawnedWorkers, 1)

	ctx = context.WithValue(ctx, log.Field("action_metadata_job_id"), strconv.Itoa(int(j.id)))

	log.Debug(ctx, "hatchery> spawnWorkerForJob> %d", j.id)
	defer log.Info(ctx, "hatchery> spawnWorkerForJob> %d (%.3f seconds elapsed)", j.id, time.Since(time.Unix(j.timestamp, 0)).Seconds())

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

	ctxQueueJobBook, next := telemetry.Span(ctx, "hatchery.QueueJobBook")
	ctxQueueJobBook, cancel := context.WithTimeout(ctxQueueJobBook, 10*time.Second)
	bookedInfos, err := h.CDSClient().QueueJobBook(ctxQueueJobBook, j.id)
	if err != nil {
		next()
		// perhaps already booked by another hatchery
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Info(ctx, "hatchery> spawnWorkerForJob> %d - cannot book job %d: %s", j.timestamp, j.id, err)
		cancel()
		return false
	}
	next()
	cancel()
	log.Debug(ctx, "hatchery> spawnWorkerForJob> %d - send book job %d", j.timestamp, j.id)

	ctxSendSpawnInfo, next := telemetry.Span(ctx, "hatchery.SendSpawnInfo", telemetry.Tag("msg", sdk.MsgSpawnInfoHatcheryStarts.ID))
	start := time.Now()
	SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
		ID: sdk.MsgSpawnInfoHatcheryStarts.ID,
		Args: []interface{}{
			h.Service().Name,
			modelName,
		},
	})
	next()

	workerName := namesgenerator.GenerateWorkerName(modelName, "")

	ctxSpawnWorker, next := telemetry.Span(ctx, "hatchery.SpawnWorker", telemetry.Tag(telemetry.TagWorker, workerName))
	arg := SpawnArguments{
		WorkerName:   workerName,
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

	ctxSpawnWorker = context.WithValue(ctxSpawnWorker, cdslog.AuthWorkerName, arg.WorkerName)
	log.Info(ctx, "starting worker %q from model %q (project: %s, workflow: %s , job:%v, jobID:%v)", arg.WorkerName, modelName, arg.ProjectKey, arg.WorkflowName, arg.NodeRunName, arg.JobID)

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
	errSpawn := h.SpawnWorker(ctxSpawnWorker, arg)
	next()
	if errSpawn != nil {
		ctx = sdk.ContextWithStacktrace(ctx, errSpawn)
		ctxSendSpawnInfo, next = telemetry.Span(ctx, "hatchery.QueueJobSendSpawnInfo", telemetry.Tag("status", "errSpawn"), telemetry.Tag("msg", sdk.MsgSpawnInfoHatcheryErrorSpawn.ID))
		SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
			ID:   sdk.MsgSpawnInfoHatcheryErrorSpawn.ID,
			Args: []interface{}{h.Service().Name, modelName, sdk.Round(time.Since(start), time.Second).String(), sdk.ExtractHTTPError(errSpawn).Error()},
		})
		log.ErrorWithStackTrace(ctx, sdk.WrapError(errSpawn, "hatchery %s cannot spawn worker %s for job %d", h.Service().Name, modelName, j.id))
		next()
		return false
	}

	if j.model != nil && j.model.IsDeprecated {
		ctxSendSpawnInfo, next = telemetry.Span(ctx, "hatchery.SendSpawnInfo", telemetry.Tag("msg", sdk.MsgSpawnInfoDeprecatedModel.ID))
		SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
			ID:   sdk.MsgSpawnInfoDeprecatedModel.ID,
			Args: []interface{}{modelName},
		})
		next()
	}
	return true // ok for this job
}
