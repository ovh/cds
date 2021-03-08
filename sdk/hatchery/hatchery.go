package hatchery

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	cache "github.com/patrickmn/go-cache"
	"github.com/rockbears/log"
	"go.opencensus.io/stats"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/telemetry"
)

var (
	// Client is a CDS Client
	Client                 cdsclient.HTTPClient
	defaultMaxProvisioning = 10
	models                 []sdk.Model
)

// Create creates hatchery
func Create(ctx context.Context, h Interface) error {
	ctx = telemetry.ContextWithTag(ctx,
		telemetry.TagServiceName, h.Name(),
		telemetry.TagServiceType, h.Type(),
	)

	if err := InitMetrics(ctx); err != nil {
		return err
	}

	// Init call hatchery.Register()
	if err := h.InitHatchery(ctx); err != nil {
		return sdk.WrapError(err, "init error")
	}

	var chanRegister, chanGetModels <-chan time.Time
	var modelType string

	hWithModels, isWithModels := h.(InterfaceWithModels)
	if isWithModels {
		// Call WorkerModel Enabled first
		var errwm error
		models, errwm = hWithModels.WorkerModelsEnabled()
		if errwm != nil {
			log.Error(ctx, "error on h.WorkerModelsEnabled() (init call): %v", errwm)
			return errwm
		}

		// using time.Tick leaks the underlying ticker but we don't care about it because it is an endless function
		chanRegister = time.Tick(time.Duration(h.Configuration().Provision.RegisterFrequency) * time.Second) // nolint
		chanGetModels = time.Tick(10 * time.Second)                                                          // nolint

		modelType = hWithModels.ModelType()
	}

	wjobs := make(chan sdk.WorkflowNodeJobRun, h.Configuration().Provision.MaxConcurrentProvisioning)
	errs := make(chan error, 1)

	// Create a cache with a default expiration time of 3 second, and which
	// purges expired items every minute
	spawnIDs := cache.New(10*time.Second, 60*time.Second)

	h.GetGoRoutines().Run(ctx, "queuePolling", func(ctx context.Context) {
		log.Debug(ctx, "starting queue polling")
		if err := h.CDSClient().QueuePolling(ctx, h.GetGoRoutines(), wjobs, errs, 20*time.Second, modelType, h.Configuration().Provision.RatioService); err != nil {
			log.Error(ctx, "Queues polling stopped: %v", err)
		}
	})

	// run the starters pool
	workersStartChan := startWorkerStarters(ctx, h)

	hostname, err := os.Hostname()
	if err != nil {
		return sdk.WrapError(err, "cannot retrieve hostname")
	}

	// read the errs channel in another goroutine too
	h.GetGoRoutines().Run(ctx, "checkErrs", func(ctx context.Context) {
		for err := range errs {
			log.Error(ctx, "%v", err)
		}
	})

	h.GetGoRoutines().Run(ctx, "mainRoutine", func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				log.Error(ctx, "cancel hatchery main routine: %v", ctx.Err())
				return

			case <-chanGetModels:
				var errwm error
				models, errwm = hWithModels.WorkerModelsEnabled()
				if errwm != nil {
					log.Error(ctx, "error on h.WorkerModelsEnabled(): %v", errwm)
				}
			case j := <-wjobs:
				t0 := time.Now()
				if j.ID == 0 {
					continue
				}
				if h.GetLogger() == nil {
					log.Error(ctx, "logger not found, don't spawn workers")
					continue
				}

				var traceEnded *struct{}
				currentCtx, currentCancel := context.WithTimeout(ctx, 10*time.Minute)
				if val, has := j.Header.Get(telemetry.SampledHeader); has && val == "1" {
					currentCtx, _ = telemetry.New(currentCtx, h, "hatchery.JobReceive", trace.AlwaysSample(), trace.SpanKindServer)

					r, _ := j.Header.Get(sdk.WorkflowRunHeader)
					w, _ := j.Header.Get(sdk.WorkflowHeader)
					p, _ := j.Header.Get(sdk.ProjectKeyHeader)

					telemetry.Current(currentCtx,
						telemetry.Tag(telemetry.TagWorkflow, w),
						telemetry.Tag(telemetry.TagWorkflowRun, r),
						telemetry.Tag(telemetry.TagProjectKey, p),
						telemetry.Tag(telemetry.TagWorkflowNodeJobRun, j.ID),
					)

					if _, ok := j.Header["WS"]; ok {
						log.Debug(ctx, "hatchery> received job from WS")
						telemetry.Current(currentCtx,
							telemetry.Tag("from", "ws"),
						)
					}
				}
				endTrace := func(reason string) {
					if reason != "" {
						telemetry.Current(currentCtx,
							telemetry.Tag("reason", reason),
						)
					}
					telemetry.End(currentCtx, nil, nil) // nolint
					var T struct{}
					traceEnded = &T
					currentCancel()
				}
				go func() {
					<-currentCtx.Done()
					if traceEnded == nil {
						endTrace(currentCtx.Err().Error())
					}
				}()

				stats.Record(currentCtx, GetMetrics().Jobs.M(1))

				if _, ok := j.Header["WS"]; ok {
					stats.Record(currentCtx, GetMetrics().JobsWebsocket.M(1))
				}

				//Check if the jobs is concerned by a pending worker creation
				if _, exist := spawnIDs.Get(strconv.FormatInt(j.ID, 10)); exist {
					log.Debug(ctx, "job %d already spawned in previous routine", j.ID)
					endTrace("already spawned")
					continue
				}

				//Before doing anything, push in cache
				spawnIDs.SetDefault(strconv.FormatInt(j.ID, 10), j.ID)

				//Check bookedBy current hatchery
				if j.BookedBy.ID != 0 {
					log.Debug(ctx, "hatchery> job %d is already booked", j.ID)
					endTrace("booked by someone")
					continue
				}

				//Check if hatchery if able to start a new worker
				if !checkCapacities(ctx, h) {
					log.Info(ctx, "hatchery %s is not able to provision new worker", h.Service().Name)
					endTrace("no capacities")
					continue
				}

				workerRequest := workerStarterRequest{
					ctx:               currentCtx,
					cancel:            endTrace,
					id:                j.ID,
					execGroups:        j.ExecGroups,
					requirements:      j.Job.Action.Requirements,
					hostname:          hostname,
					timestamp:         time.Now().Unix(),
					workflowNodeRunID: j.WorkflowNodeRunID,
				}

				// Check at least one worker model can match
				var chosenModel *sdk.Model
				var canTakeJob bool

				var containsRegionRequirement bool
			loopRequirements:
				for _, r := range workerRequest.requirements {
					switch r.Type {
					case sdk.RegionRequirement:
						containsRegionRequirement = true
						break loopRequirements
					}
				}

				if !containsRegionRequirement && h.Configuration().Provision.IgnoreJobWithNoRegion {
					log.Debug(ctx, "cannot launch this job because it does not contains a region prerequisite and IgnoreJobWithNoRegion=true in hatchery configuration")
					canTakeJob = false
				} else if isWithModels {
					for i := range models {
						if canRunJobWithModel(ctx, hWithModels, workerRequest, &models[i]) {
							chosenModel = &models[i]
							canTakeJob = true
							break
						}
					}

					// No model has been found, let's send a failing result
					if chosenModel == nil {
						log.Debug(ctx, "hatchery> no model")
						endTrace("no model")
						continue
					}
				} else {
					if canRunJob(ctx, h, workerRequest) {
						log.Debug(ctx, "hatchery %s can try to spawn a worker for job %d", h.Name(), j.ID)
						canTakeJob = true
					}
				}

				if !canTakeJob {
					log.Info(ctx, "hatchery %s is not able to run the job %d", h.Name(), j.ID)
					endTrace("cannot run job")
					continue
				}

				if chosenModel != nil {
					// We got a model, let's start a worker
					workerRequest.model = chosenModel

					// Interpolate model secrets
					if err := ModelInterpolateSecrets(hWithModels, chosenModel); err != nil {
						log.Error(ctx, "%v", err)
						continue
					}
				}

				//Ask to start
				log.Debug(ctx, "hatchery> Request a worker for job %d (%.3f seconds elapsed)", j.ID, time.Since(t0).Seconds())
				workersStartChan <- workerRequest

			case <-chanRegister:
				if err := workerRegister(ctx, hWithModels, workersStartChan); err != nil {
					log.Warn(ctx, "error on workerRegister: %v", err)
				}
			}
		}
	})
	return nil
}

func canRunJob(ctx context.Context, h Interface, j workerStarterRequest) bool {
	for _, r := range j.requirements {
		// If requirement is an hostname requirement, it's for a specific worker
		if r.Type == sdk.HostnameRequirement && r.Value != j.hostname {
			log.Debug(ctx, "canRunJob> %d - job %d - hostname requirement r.Value(%s) != hostname(%s)", j.timestamp, j.id, r.Value, j.hostname)
			return false
		}

		if r.Type == sdk.RegionRequirement && r.Value != h.Configuration().Provision.Region {
			log.Debug(ctx, "canRunJob> %d - job %d - job with region requirement: cannot spawn. hatchery-region:%s prerequisite:%s", j.timestamp, j.id, h.Configuration().Provision.Region, r.Value)
			return false
		}

		// Skip others requirement as we can't check it
		if r.Type == sdk.PluginRequirement || r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			log.Debug(ctx, "canRunJob> %d - job %d - job with service, plugin or memory requirement. Skip these check as we can't check it on hatchery routine", j.timestamp, j.id)
			continue
		}

	}
	return h.CanSpawn(ctx, nil, j.id, j.requirements)
}

// MemoryRegisterContainer is the RAM used for spawning
// a docker container for register a worker model. 128 Mo
const MemoryRegisterContainer int64 = 128

func canRunJobWithModel(ctx context.Context, h InterfaceWithModels, j workerStarterRequest, model *sdk.Model) bool {
	if model.Type != h.ModelType() {
		log.Debug(ctx, "canRunJobWithModel> model %s type:%s current hatchery modelType: %s", model.Name, model.Type, h.ModelType())
		return false
	}

	// If the model needs registration, don't spawn for now
	if h.NeedRegistration(ctx, model) {
		log.Debug(ctx, "canRunJobWithModel> model %s needs registration", model.Name)
		return false
	}

	if model.NbSpawnErr > 5 {
		log.Warn(ctx, "canRunJobWithModel> Too many errors on spawn with model %s, please check this worker model", model.Name)
		return false
	}

	if len(j.execGroups) > 0 {
		checkGroup := false
		for _, g := range j.execGroups {
			if g.ID == model.GroupID {
				checkGroup = true
				break
			}
		}
		if !checkGroup {
			log.Debug(ctx, "canRunJobWithModel> job %d - model %s attached to group %d can't run this job", j.id, model.Name, model.GroupID)
			return false
		}
	}

	var containsModelRequirement, containsHostnameRequirement bool
	for _, r := range j.requirements {
		switch r.Type {
		case sdk.ModelRequirement:
			containsModelRequirement = true
		case sdk.HostnameRequirement:
			containsHostnameRequirement = true
		}
	}

	if model.IsDeprecated && !containsModelRequirement {
		log.Debug(ctx, "canRunJobWithModel> %d - job %d - Cannot launch this model because it is deprecated", j.timestamp, j.id)
		return false
	}

	// Common check
	for _, r := range j.requirements {
		// If requirement is a Model requirement, it's easy. It's either can or can't run
		// r.Value could be: theModelName --port=8888:9999, so we take strings.Split(r.Value, " ")[0] to compare
		// only modelName
		if r.Type == sdk.ModelRequirement {
			modelName := strings.Split(r.Value, " ")[0]
			isGroupModel := modelName == fmt.Sprintf("%s/%s", model.Group.Name, model.Name)
			isSharedInfraModel := model.Group.Name == sdk.SharedInfraGroupName && modelName == model.Name
			isSameName := modelName == model.Name // for backward compatibility with runs, if only the name match we considered that the model can be used, keep this condition until the workflow runs were not migrated.
			if !isGroupModel && !isSharedInfraModel && !isSameName {
				log.Debug(ctx, "canRunJobWithModel> %d - job %d - model requirement r.Value(%s) do not match model.Name(%s) and model.Group(%s)", j.timestamp, j.id, strings.Split(r.Value, " ")[0], model.Name, model.Group.Name)
				return false
			}
		}

		// service and memory requirements are only supported by docker model
		if model.Type != sdk.Docker && (r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement) {
			log.Debug(ctx, "canRunJobWithModel> %d - job %d - job with service requirement or memory requirement: only for model docker. current model:%s", j.timestamp, j.id, model.Type)
			return false
		}

		// Skip other requirement as we can't check it
		if r.Type == sdk.PluginRequirement || r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			log.Debug(ctx, "canRunJobWithModel> %d - job %d - job with service, plugin, network or memory requirement. Skip these check as we can't check it on hatchery routine", j.timestamp, j.id)
			continue
		}

		if r.Type == sdk.OSArchRequirement && model.RegisteredOS != nil && *model.RegisteredOS != "" && model.RegisteredArch != nil && *model.RegisteredArch != "" && r.Value != (*model.RegisteredOS+"/"+*model.RegisteredArch) {
			log.Debug(ctx, "canRunJobWithModel> %d - job %d - job with OSArch requirement: cannot spawn on this OSArch. current model: %s/%s", j.timestamp, j.id, *model.RegisteredOS, *model.RegisteredArch)
			return false
		}

		if r.Type == sdk.RegionRequirement && r.Value != h.Configuration().Provision.Region {
			log.Debug(ctx, "canRunJobWithModel> %d - job %d - job with region requirement: cannot spawn. hatchery-region:%s prerequisite:%s", j.timestamp, j.id, h.Configuration().Provision.Region, r.Value)
			return false
		}

		if !containsModelRequirement && !containsHostnameRequirement {
			if r.Type == sdk.BinaryRequirement {
				found := false
				// Check binary requirement against worker model capabilities
				for _, c := range model.RegisteredCapabilities {
					if r.Value == c.Value || r.Value == c.Name {
						found = true
						break
					}
				}

				if !found {
					log.Debug(ctx, "canRunJobWithModel> %d - job %d - model(%s) does not have binary %s(%s) for this job.", j.timestamp, j.id, model.Name, r.Name, r.Value)
					return false
				}
			}
		}
	}

	return h.CanSpawn(ctx, model, j.id, j.requirements)
}

// SendSpawnInfo sends a spawnInfo
func SendSpawnInfo(ctx context.Context, h Interface, jobID int64, spawnMsg sdk.SpawnMsg) {
	if h.CDSClient() == nil || jobID == 0 {
		return
	}
	infos := []sdk.SpawnInfo{{RemoteTime: time.Now(), Message: spawnMsg, UserMessage: spawnMsg.DefaultUserMessage()}}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := h.CDSClient().QueueJobSendSpawnInfo(ctx, jobID, infos); err != nil {
		log.Warn(ctx, "SendSpawnInfo> cannot client.sendSpawnInfo for job %d: %s", jobID, err)
	}
}
