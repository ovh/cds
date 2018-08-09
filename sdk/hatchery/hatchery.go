package hatchery

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/tracingutils"
)

var (
	// Client is a CDS Client
	Client                 sdk.HTTPClient
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
		tag.Upsert(TagHatcheryName, h.Hatchery().Name),
	)
	return ctx
}

// Create creates hatchery
func Create(h Interface) error {
	ctx := context.Background()
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

	if err := h.Init(); err != nil {
		return fmt.Errorf("Create> Init error: %v", err)
	}

	// Register the current hatchery to be sure it's authentifed on CDS API before doing any call
	if err := Register(h); err != nil {
		return fmt.Errorf("Create> Register error: %v", err)
	}

	// Call WorkerModel Enabled first
	var errwm error
	models, errwm = h.CDSClient().WorkerModelsEnabled()
	if errwm != nil {
		log.Error("error on h.CDSClient().WorkerModelsEnabled() (init call): %v", errwm)
	}

	tickerProvision := time.NewTicker(time.Duration(h.Configuration().Provision.Frequency) * time.Second)
	tickerRegister := time.NewTicker(time.Duration(h.Configuration().Provision.RegisterFrequency) * time.Second)
	tickerGetModels := time.NewTicker(10 * time.Second)

	defer func() {
		tickerProvision.Stop()
		tickerRegister.Stop()
		tickerGetModels.Stop()
	}()

	pbjobs := make(chan sdk.PipelineBuildJob, 10)
	wjobs := make(chan sdk.WorkflowNodeJobRun, 10)
	errs := make(chan error, 1)

	// Create a cache with a default expiration time of 3 second, and which
	// purges expired items every minute
	spawnIDs := cache.New(10*time.Second, 60*time.Second)

	// hatchery is now fully Initialized
	h.SetInitialized()

	sdk.GoRoutine("heartbeat", func() {
		hearbeat(h, h.Configuration().API.Token, h.Configuration().API.MaxHeartbeatFailures)
	})

	sdk.GoRoutine("queuePolling", func() {
		if err := h.CDSClient().QueuePolling(ctx, wjobs, pbjobs, errs, 2*time.Second, h.Configuration().Provision.GraceTimeQueued, nil); err != nil {
			log.Error("Queues polling stopped: %v", err)
			cancel()
		}
	})

	// run the starters pool
	workersStartChan, workerStartResultChan := startWorkerStarters(h)

	hostname, errh := os.Hostname()
	if errh != nil {
		return fmt.Errorf("Create> Cannot retrieve hostname: %s", errh)
	}
	// read the result channel in another goroutine to let the main goroutine start new workers
	sdk.GoRoutine("checkStarterResult", func() {
		for startWorkerRes := range workerStartResultChan {
			if startWorkerRes.err != nil {
				errs <- startWorkerRes.err
			}
			if startWorkerRes.temptToSpawn {
				found := false
				for _, hID := range startWorkerRes.request.spawnAttempts {
					if hID == h.ID() {
						found = true
						break
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
			t0 := time.Now()
			if j.ID == 0 {
				continue
			}

			//Check spawnsID
			if _, exist := spawnIDs.Get(strconv.FormatInt(j.ID, 10)); exist {
				log.Debug("pipeline build job %d already spawned in previous routine", j.ID)
				continue
			}

			//Before doing anything, push in cache
			spawnIDs.SetDefault(strconv.FormatInt(j.ID, 10), j.ID)

			//Check bookedBy current hatchery
			if j.BookedBy.ID != 0 && j.BookedBy.ID != h.ID() {
				continue
			}

			//Check gracetime
			if j.QueuedSeconds < int64(h.Configuration().Provision.GraceTimeQueued) {
				log.Debug("pipeline build  job %d is too fresh, queued since %d seconds, let existing waiting worker check it", j.ID, j.QueuedSeconds)
				continue
			}

			//Check if hatchery if able to start a new worker
			if !checkCapacities(h) {
				log.Info("hatchery %s is not able to provision new worker", h.Hatchery().Name)
				continue
			}

			//Ask to start
			workerRequest := workerStarterRequest{
				id:            j.ID,
				isWorkflowJob: false,
				execGroups:    j.ExecGroups,
				requirements:  j.Job.Action.Requirements,
				hostname:      hostname,
				timestamp:     time.Now().Unix(),
			}

			// Check at least one worker model can match
			var chosenModel *sdk.Model
			for i := range models {
				if canRunJob(h, workerRequest, models[i]) {
					chosenModel = &models[i]
					break
				}
			}

			if chosenModel == nil {
				//do something
				continue
			}

			workerRequest.model = *chosenModel
			log.Info("hatchery> Request a worker for pipeline build job %d (%.3f seconds elapsed)", j.ID, time.Since(t0).Seconds())
			workersStartChan <- workerRequest

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

			stats.Record(currentCtx, h.Stats().Jobs.M(1))

			//Check if the jobs is concerned by a pending worker creation
			if _, exist := spawnIDs.Get(strconv.FormatInt(j.ID, 10)); exist {
				log.Debug("job %d already spawned in previous routine", j.ID)
				endTrace("already spawned")
				continue
			}

			//Check gracetime
			if j.QueuedSeconds < int64(h.Configuration().Provision.GraceTimeQueued) {
				log.Debug("job %d is too fresh, queued since %d seconds, let existing waiting worker check it", j.ID)
				endTrace("too fresh")
				continue
			}

			//Before doing anything, push in cache
			spawnIDs.SetDefault(strconv.FormatInt(j.ID, 10), j.ID)

			//Check bookedBy current hatchery
			if j.BookedBy.ID != 0 && j.BookedBy.ID != h.ID() {
				log.Debug("hatchery> job %d is booked by someone else (%d / %d)", j.ID, j.BookedBy.ID, h.ID())
				endTrace("booked by someone else")
				continue
			}

			//Check if hatchery if able to start a new worker
			if !checkCapacities(h) {
				log.Info("hatchery %s is not able to provision new worker", h.Hatchery().Name)
				endTrace("no capacities")
				continue
			}

			workerRequest := workerStarterRequest{
				ctx:               currentCtx,
				cancel:            endTrace,
				id:                j.ID,
				isWorkflowJob:     true,
				execGroups:        j.ExecGroups,
				requirements:      j.Job.Action.Requirements,
				hostname:          hostname,
				timestamp:         time.Now().Unix(),
				spawnAttempts:     j.SpawnAttempts,
				workflowNodeRunID: j.WorkflowNodeRunID,
			}

			// Check at least one worker model can match
			var chosenModel *sdk.Model
			for i := range models {
				if canRunJob(h, workerRequest, models[i]) {
					chosenModel = &models[i]
					break
				}
			}

			// No model has been found, let's send a failing result
			if chosenModel == nil {
				workerStartResultChan <- workerStarterResult{
					request:      workerRequest,
					isRun:        false,
					temptToSpawn: true,
				}
				endTrace("no model or service ratio reached")
				continue
			}

			//We got a model, let's start a worker
			workerRequest.model = *chosenModel

			//Ask to start
			log.Info("hatchery> Request a worker for job %d (%.3f seconds elapsed)", j.ID, time.Since(t0).Seconds())
			workersStartChan <- workerRequest

		case err := <-errs:
			log.Error("%v", err)

		case <-tickerProvision.C:
			provisioning(h, models)

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

func canRunJob(h Interface, j workerStarterRequest, model sdk.Model) bool {
	if model.Type != h.ModelType() {
		return false
	}

	// If the model needs registration, don't spawn for now
	if h.NeedRegistration(&model) {
		log.Debug("canRunJob> model %s needs registration", model.Name)
		return false
	}

	// if current hatchery is in same group than worker model -> do not avoid spawn, even if worker model is in error
	if model.NbSpawnErr > 5 && h.Hatchery().GroupID != model.ID {
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
	// Common check
	for _, r := range j.requirements {
		// If requirement is a Model requirement, it's easy. It's either can or can't run
		// r.Value could be: theModelName --port=8888:9999, so we take strings.Split(r.Value, " ")[0] to compare
		// only modelName
		if r.Type == sdk.ModelRequirement && strings.Split(r.Value, " ")[0] != model.Name {
			log.Debug("canRunJob> %d - job %d - model requirement r.Value(%s) != model.Name(%s)", j.timestamp, j.id, strings.Split(r.Value, " ")[0], model.Name)
			return false
		}

		// If requirement is an hostname requirement, it's for a specific worker
		if r.Type == sdk.HostnameRequirement && r.Value != j.hostname {
			log.Debug("canRunJob> %d - job %d - hostname requirement r.Value(%s) != hostname(%s)", j.timestamp, j.id, r.Value, j.hostname)
			return false
		}

		// service and memory requirements are only supported by docker model
		if model.Type != sdk.Docker && (r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement) {
			log.Debug("canRunJob> %d - job %d - job with service requirement or memory requirement: only for model docker. current model:%s", j.timestamp, j.id, model.Type)
			return false
		}

		// Skip network access requirement as we can't check it
		if r.Type == sdk.NetworkAccessRequirement || r.Type == sdk.PluginRequirement || r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			log.Debug("canRunJob> %d - job %d - job with service requirement or memory requirement: only for model docker. current model:%s", j.timestamp, j.id, model.Type)
			continue
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

	return h.CanSpawn(&model, j.id, j.requirements)
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
