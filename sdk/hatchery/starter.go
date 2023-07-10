package hatchery

import (
	"context"
	"fmt"
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
	cancel              func()
	id                  string
	model               sdk.WorkerStarterWorkerModel
	execGroups          []sdk.Group
	requirements        []sdk.Requirement
	hostname            string
	workflowNodeRunID   int64
	registerWorkerModel *sdk.Model
	queued              time.Time
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
			j.cancel() // call to EndTrace for observability
		} else { // Start a worker for registering
			log.Debug(ctx, "Spawning worker for register model %s", m.Name)
			if atomic.LoadInt64(&nbWorkerToStart) > int64(h.Configuration().Provision.MaxConcurrentProvisioning) {
				j.cancel() // call to EndTrace for observability
				continue
			}

			workerName := namesgenerator.GenerateWorkerName(m.Name, "register")

			atomic.AddInt64(&nbWorkerToStart, 1)
			// increment nbRegisteringWorkerModels, but no decrement.
			// this counter is reset with func workerRegister
			atomic.AddInt64(&nbRegisteringWorkerModels, 1)
			arg := SpawnArguments{
				WorkerName:   workerName,
				Model:        sdk.WorkerStarterWorkerModel{ModelV1: m},
				RegisterOnly: true,
				JobID:        "0",
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
				j.cancel() // call to EndTrace for observability
				continue
			}
			arg.WorkerToken = jwt

			if err := h.SpawnWorker(j.ctx, arg); err != nil {
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
			j.cancel() // call to EndTrace for observability
		}
	}
}

func spawnWorkerForJob(ctx context.Context, h Interface, j workerStarterRequest) bool {
	ctx, end := telemetry.Span(ctx, "hatchery.spawnWorkerForJob", telemetry.Tag(telemetry.TagWorkflowNodeJobRun, j.id))
	defer end()

	ctx = telemetry.ContextWithTag(ctx,
		telemetry.TagServiceName, h.Name(),
		telemetry.TagServiceType, h.Type(),
	)
	telemetry.Record(ctx, GetMetrics().SpawnedWorkers, 1)

	logStepInfo(ctx, "starting-worker", j.queued)

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
	if j.model.ModelV1 != nil {
		modelName = j.model.GetFullPath()
	} else if j.model.ModelV2 != nil {
		modelName = j.model.GetName()
	}
	ctx = context.WithValue(ctx, LogFieldModel, modelName)

	arg := SpawnArguments{
		WorkerName:   namesgenerator.GenerateWorkerName(modelName, ""),
		Model:        j.model,
		JobID:        j.id,
		NodeRunID:    j.workflowNodeRunID,
		Requirements: j.requirements,
		HatcheryName: h.Service().Name,
	}

	if sdk.IsValidUUID(j.id) {
		jobRun, err := h.CDSClientV2().V2HatcheryTakeJob(ctx, j.id)
		if err != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Info(ctx, "cannot book job: %s", err)
			return false
		}
		arg.RunID = jobRun.WorkflowRunID
		arg.WorkflowName = jobRun.WorkflowName
		arg.ProjectKey = jobRun.ProjectKey
		arg.JobName = jobRun.JobID
	} else {
		if h.Service() == nil {
			log.Warn(ctx, "hatchery not registered")
			return false
		}
		ctxQueueJobBook, next := telemetry.Span(ctx, "hatchery.QueueJobBook")
		ctxQueueJobBook, cancel := context.WithTimeout(ctxQueueJobBook, 10*time.Second)
		bookedInfos, err := h.CDSClient().QueueJobBook(ctxQueueJobBook, j.id)
		if err != nil {
			next()
			// perhaps already booked by another hatchery
			ctx = sdk.ContextWithStacktrace(ctx, err)
			log.Info(ctx, "cannot book job: %s", err)
			cancel()
			return false
		}
		next()
		cancel()

		ctxSendSpawnInfo, next := telemetry.Span(ctx, "hatchery.SendSpawnInfo", telemetry.Tag("msg", sdk.MsgSpawnInfoHatcheryStarts.ID))
		SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
			ID: sdk.MsgSpawnInfoHatcheryStarts.ID,
			Args: []interface{}{
				h.Service().Name,
				modelName,
			},
		})
		next()

		arg.NodeRunName = bookedInfos.NodeRunName
		arg.RunID = fmt.Sprintf("%d", bookedInfos.RunID)
		arg.WorkflowID = bookedInfos.WorkflowID
		arg.WorkflowName = bookedInfos.WorkflowName
		arg.ProjectKey = bookedInfos.ProjectKey
		arg.JobName = bookedInfos.JobName
	}

	logStepInfo(ctx, "book-job", j.queued)
	start := time.Now()

	ctx = context.WithValue(ctx, cdslog.AuthWorkerName, arg.WorkerName)
	ctx = context.WithValue(ctx, LogFieldProject, arg.ProjectKey)
	ctx = context.WithValue(ctx, LogFieldWorkflow, arg.WorkflowName)
	ctx = context.WithValue(ctx, LogFieldNodeRun, arg.NodeRunName)

	var serviceCount int
	for i := range arg.Requirements {
		if arg.Requirements[i].Type == sdk.ServiceRequirement {
			serviceCount++
		}
	}
	ctx = context.WithValue(ctx, LogFieldServiceCount, serviceCount)

	// Get a JWT to authentified the worker
	jwt, err := NewWorkerToken(h.Service().Name, h.GetPrivateKey(), time.Now().Add(1*time.Hour), arg)
	if err != nil {
		if j.model.ModelV1 != nil {
			ctx = sdk.ContextWithStacktrace(ctx, err)
			var spawnError = sdk.SpawnErrorForm{
				Error: fmt.Sprintf("cannot spawn worker for register: %v", err),
			}
			if err := h.CDSClient().WorkerModelSpawnError(j.model.ModelV1.Group.Name, j.model.ModelV1.Name, spawnError); err != nil {
				log.Error(ctx, "error on call client.WorkerModelSpawnError on worker model %s for register: %s", j.model.ModelV1.Name, err)
			}
		} else if j.model.ModelV2 != nil {
			if err := h.CDSClientV2().V2QueueJobResult(ctx, arg.JobID, sdk.V2WorkflowRunJobResult{
				Status: sdk.StatusFail,
				Error:  "unable to generate worker token",
			}); err != nil {
				log.ErrorWithStackTrace(ctx, err)
				return false
			}
		}
		// TODO manage modelv2
		return false
	}
	arg.WorkerToken = jwt

	logStepInfo(ctx, "starting-worker-spawn", j.queued)

	ctxSpawnWorker, next := telemetry.Span(ctx, "hatchery.SpawnWorker", telemetry.Tag(telemetry.TagWorker, arg.WorkerName))
	errSpawn := h.SpawnWorker(ctxSpawnWorker, arg)
	next()
	if errSpawn != nil {
		ctx = sdk.ContextWithStacktrace(ctx, errSpawn)
		ctxSendSpawnInfo, next := telemetry.Span(ctx, "hatchery.QueueJobSendSpawnInfo", telemetry.Tag("status", "errSpawn"), telemetry.Tag("msg", sdk.MsgSpawnInfoHatcheryErrorSpawn.ID))
		SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
			ID:   sdk.MsgSpawnInfoHatcheryErrorSpawn.ID,
			Args: []interface{}{h.Service().Name, modelName, sdk.Round(time.Since(start), time.Second).String(), sdk.ExtractHTTPError(errSpawn).Error()},
		})
		log.ErrorWithStackTrace(ctx, sdk.WrapError(errSpawn, "hatchery %s cannot spawn worker %s for job %s", h.Service().Name, modelName, j.id))
		next()
		return false
	}

	if j.model.ModelV1 != nil && j.model.ModelV1.IsDeprecated {
		ctxSendSpawnInfo, next := telemetry.Span(ctx, "hatchery.SendSpawnInfo", telemetry.Tag("msg", sdk.MsgSpawnInfoDeprecatedModel.ID))
		SendSpawnInfo(ctxSendSpawnInfo, h, j.id, sdk.SpawnMsg{
			ID:   sdk.MsgSpawnInfoDeprecatedModel.ID,
			Args: []interface{}{modelName},
		})
		next()
	}

	return true // ok for this job
}
