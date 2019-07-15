package hatchery

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	cache "github.com/patrickmn/go-cache"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/tracingutils"
)

var (
	// Client is a CDS Client
	Client                 cdsclient.HTTPClient
	defaultMaxProvisioning = 10
	models                 []sdk.Model

	// Opencensus tags
	TagHatchery     tag.Key
	TagHatcheryName tag.Key
)

func init() {
	TagHatchery, _ = tag.NewKey("hatchery")
	TagHatcheryName, _ = tag.NewKey("hatchery_name")
}

// WithTags returns a context with opencenstus tags
func WithTags(ctx context.Context, h Interface) context.Context {
	ctx, _ = tag.New(ctx,
		tag.Upsert(TagHatchery, h.ServiceName()),
		tag.Upsert(TagHatcheryName, h.Service().Name),
	)
	return ctx
}

// Create creates hatchery
func Create(ctx context.Context, h Interface) error {
	ctx, cancel := context.WithCancel(ctx)

	// Gracefully shutdown connections
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
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

	// Init call hatchery.Register()
	if err := h.InitHatchery(); err != nil {
		return fmt.Errorf("Create> Init error: %v", err)
	}

	var chanRegister, chanGetModels, chanProvision <-chan time.Time
	var modelType string

	hWithModels, isWithModels := h.(InterfaceWithModels)
	if isWithModels {
		// Call WorkerModel Enabled first
		var errwm error
		models, errwm = hWithModels.WorkerModelsEnabled()
		if errwm != nil {
			log.Error("error on h.WorkerModelsEnabled() (init call): %v", errwm)
			return errwm
		}

		chanRegister = time.Tick(time.Duration(h.Configuration().Provision.RegisterFrequency) * time.Second)
		chanGetModels = time.Tick(10 * time.Second)
		chanProvision = time.Tick(time.Duration(h.Configuration().Provision.Frequency) * time.Second)

		modelType = hWithModels.ModelType()
	}

	wjobs := make(chan sdk.WorkflowNodeJobRun, h.Configuration().Provision.MaxConcurrentProvisioning)
	errs := make(chan error, 1)

	// Create a cache with a default expiration time of 3 second, and which
	// purges expired items every minute
	spawnIDs := cache.New(10*time.Second, 60*time.Second)

	sdk.GoRoutine(ctx, "queuePolling",
		func(ctx context.Context) {
			if err := h.CDSClient().QueuePolling(ctx, wjobs, errs, 20*time.Second, modelType, h.Hatchery().RatioService); err != nil {
				log.Error("Queues polling stopped: %v", err)
				cancel()
			}
		},
		PanicDump(h),
	)

	// run the starters pool
	workersStartChan := startWorkerStarters(ctx, h)

	hostname, errh := os.Hostname()
	if errh != nil {
		return fmt.Errorf("Create> Cannot retrieve hostname: %s", errh)
	}

	// read the errs channel in another goroutine too
	sdk.GoRoutine(ctx, "checkErrs", func(ctx context.Context) {
		for err := range errs {
			log.Error("%v", err)
		}
	}, PanicDump(h))

	// the main goroutine
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-chanGetModels:
			var errwm error
			models, errwm = hWithModels.WorkerModelsEnabled()
			if errwm != nil {
				log.Error("error on h.WorkerModelsEnabled(): %v", errwm)
			}
		case j := <-wjobs:
			t0 := time.Now()
			if j.ID == 0 {
				continue
			}

			var traceEnded *struct{}
			currentCtx, currentCancel := context.WithTimeout(ctx, 10*time.Minute)
			currentCtx = WithTags(currentCtx, h)
			if val, has := j.Header.Get(tracingutils.SampledHeader); has && val == "1" {
				currentCtx, _ = observability.New(currentCtx, h.ServiceName(), "hatchery.JobReceive", trace.AlwaysSample(), trace.SpanKindServer)

				r, _ := j.Header.Get(sdk.WorkflowRunHeader)
				w, _ := j.Header.Get(sdk.WorkflowHeader)
				p, _ := j.Header.Get(sdk.ProjectKeyHeader)

				observability.Current(currentCtx,
					observability.Tag(observability.TagWorkflow, w),
					observability.Tag(observability.TagWorkflowRun, r),
					observability.Tag(observability.TagProjectKey, p),
					observability.Tag(observability.TagWorkflowNodeJobRun, j.ID),
				)

				if _, ok := j.Header["SSE"]; ok {
					log.Debug("hatchery> received job from SSE")
					observability.Current(currentCtx,
						observability.Tag("from", "sse"),
					)
				}
			}
			endTrace := func(reason string) {
				if reason != "" {
					observability.Current(currentCtx,
						observability.Tag("reason", reason),
					)
				}
				observability.End(currentCtx, nil, nil) // nolint
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

			stats.Record(currentCtx, h.Metrics().Jobs.M(1))

			if _, ok := j.Header["SSE"]; ok {
				stats.Record(currentCtx, h.Metrics().JobsSSE.M(1))
			}

			//Check if the jobs is concerned by a pending worker creation
			if _, exist := spawnIDs.Get(strconv.FormatInt(j.ID, 10)); exist {
				log.Debug("job %d already spawned in previous routine", j.ID)
				endTrace("already spawned")
				continue
			}

			//Before doing anything, push in cache
			spawnIDs.SetDefault(strconv.FormatInt(j.ID, 10), j.ID)

			//Check bookedBy current hatchery
			if j.BookedBy.ID != 0 {
				log.Debug("hatchery> job %d is booked by someone", j.ID)
				endTrace("booked by someone")
				continue
			}

			//Check if hatchery if able to start a new worker
			if !checkCapacities(ctx, h) {
				log.Info("hatchery %s is not able to provision new worker", h.Service().Name)
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
			if isWithModels {
				for i := range models {
					if canRunJobWithModel(hWithModels, workerRequest, &models[i]) {
						chosenModel = &models[i]
						canTakeJob = true
						break
					}
				}

				// No model has been found, let's send a failing result
				if chosenModel == nil {
					log.Debug("hatchery> no model")
					endTrace("no model")
					continue
				}
			} else {
				if canRunJob(h, workerRequest) {
					log.Debug("hatchery %s can try to spawn a worker for job %d", h.ServiceName(), j.ID)
					canTakeJob = true
				}
			}

			if !canTakeJob {
				log.Info("hatchery %s is not able to run the job %d", h.ServiceName(), j.ID)
				endTrace("cannot run job")
				continue
			}

			if chosenModel != nil {
				//We got a model, let's start a worker
				workerRequest.model = chosenModel
			}

			//Ask to start
			log.Debug("hatchery> Request a worker for job %d (%.3f seconds elapsed)", j.ID, time.Since(t0).Seconds())
			workersStartChan <- workerRequest

		case <-chanProvision:
			provisioning(hWithModels, models)

		case <-chanRegister:
			if err := workerRegister(ctx, hWithModels, workersStartChan); err != nil {
				log.Warning("Error on workerRegister: %s", err)
			}
		}
	}
}

func canRunJob(h Interface, j workerStarterRequest) bool {
	for _, r := range j.requirements {
		// If requirement is an hostname requirement, it's for a specific worker
		if r.Type == sdk.HostnameRequirement && r.Value != j.hostname {
			log.Debug("canRunJob> %d - job %d - hostname requirement r.Value(%s) != hostname(%s)", j.timestamp, j.id, r.Value, j.hostname)
			return false
		}

		// Skip network access requirement as we can't check it
		if r.Type == sdk.NetworkAccessRequirement || r.Type == sdk.PluginRequirement || r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			log.Debug("canRunJob> %d - job %d - job with service, plugin, network or memory requirement. Skip these check as we can't checkt it on hatchery routine", j.timestamp, j.id)
			continue
		}
	}
	return h.CanSpawn(nil, j.id, j.requirements)
}

// MemoryRegisterContainer is the RAM used for spawning
// a docker container for register a worker model. 128 Mo
const MemoryRegisterContainer int64 = 128

func canRunJobWithModel(h InterfaceWithModels, j workerStarterRequest, model *sdk.Model) bool {
	if model.Type != h.ModelType() {
		log.Debug("canRunJob> model %s type:%s current hatchery modelType: %s", model.Name, model.Type, h.ModelType())
		return false
	}

	// If the model needs registration, don't spawn for now
	if h.NeedRegistration(model) {
		log.Debug("canRunJob> model %s needs registration", model.Name)
		return false
	}

	if model.NbSpawnErr > 5 {
		log.Warning("canRunJob> Too many errors on spawn with model %s, please check this worker model", model.Name)
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
			log.Debug("canRunJob> job %d - model %s attached to group %d can't run this job", j.id, model.Name, model.GroupID)
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
		log.Debug("canRunJob> %d - job %d - Cannot launch this model because it is deprecated", j.timestamp, j.id)
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
				log.Debug("canRunJob> %d - job %d - model requirement r.Value(%s) do not match model.Name(%s) and model.Group(%s)", j.timestamp, j.id, strings.Split(r.Value, " ")[0], model.Name, model.Group.Name)
				return false
			}
		}

		// service and memory requirements are only supported by docker model
		if model.Type != sdk.Docker && (r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement) {
			log.Debug("canRunJob> %d - job %d - job with service requirement or memory requirement: only for model docker. current model:%s", j.timestamp, j.id, model.Type)
			return false
		}

		if r.Type == sdk.OSArchRequirement && model.RegisteredOS != "" && model.RegisteredArch != "" && r.Value != (model.RegisteredOS+"/"+model.RegisteredArch) {
			log.Debug("canRunJob> %d - job %d - job with OSArch requirement: cannot spawn on this OSArch. current model: %s/%s", j.timestamp, j.id, model.RegisteredOS, model.RegisteredArch)
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
					log.Debug("canRunJob> %d - job %d - model(%s) does not have binary %s(%s) for this job.", j.timestamp, j.id, model.Name, r.Name, r.Value)
					return false
				}
			}
		}
	}

	return h.CanSpawn(model, j.id, j.requirements)
}

// SendSpawnInfo sends a spawnInfo
func SendSpawnInfo(ctx context.Context, h Interface, jobID int64, spawnMsg sdk.SpawnMsg) {
	if h.CDSClient() == nil {
		return
	}
	infos := []sdk.SpawnInfo{{RemoteTime: time.Now(), Message: spawnMsg}}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := h.CDSClient().QueueJobSendSpawnInfo(ctx, jobID, infos); err != nil {
		log.Warning("spawnWorkerForJob> cannot client.sendSpawnInfo for job %d: %s", jobID, err)
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
