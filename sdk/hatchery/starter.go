package hatchery

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type workerStarterRequest struct {
	id                  int64
	isWorkflowJob       bool
	models              []sdk.Model
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

// Start all goroutines which manage the hatchery worker spawning routine.
// the purpose is to avoid go routines leak when there is a bunch of worker to start
func startWorkerStarters(h Interface) (chan<- workerStarterRequest, <-chan workerStarterResult) {
	jobs := make(chan workerStarterRequest, 1)
	results := make(chan workerStarterResult, 1)

	maxProv := h.Configuration().Provision.MaxConcurrentProvisioning
	if maxProv < 1 {
		maxProv = defaultMaxProvisioning
	}
	for i := 0; i < maxProv; i++ {
		sdk.GoRoutine("workerStarter", func() {
			workerStarter(h, jobs, results)
		})
	}

	return jobs, results
}

func workerStarter(h Interface, jobs <-chan workerStarterRequest, results chan<- workerStarterResult) {
	for j := range jobs {
		// Start a worker for a job
		if m := j.registerWorkerModel; m == nil {
			//Try to start the worker
			isRun, err := spawnWorkerForJob(h, j)
			//Check the result
			res := workerStarterResult{
				request:      j,
				err:          err,
				isRun:        isRun,
				temptToSpawn: true,
			}
			//Send the result back
			results <- res

		} else { // Start a worker for registering
			log.Debug("Spawning worker for register model %s", m.Name)
			if atomic.LoadInt64(&nbWorkerToStart) > int64(h.Configuration().Provision.MaxConcurrentProvisioning) {
				continue
			}

			atomic.AddInt64(&nbWorkerToStart, 1)
			atomic.AddInt64(&nbRegisteringWorkerModels, 1)
			if _, errSpawn := h.SpawnWorker(SpawnArguments{Model: *m, IsWorkflowJob: false, JobID: 0, Requirements: nil, RegisterOnly: true, LogInfo: "spawn for register"}); errSpawn != nil {
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
	log.Debug("spawnWorkerForJob> %d", j.id)
	defer logTime(h, fmt.Sprintf("spawnWorkerForJob> %d elapsed", j.timestamp), time.Now())
	maxProv := h.Configuration().Provision.MaxConcurrentProvisioning
	if maxProv < 1 {
		maxProv = defaultMaxProvisioning
	}
	if atomic.LoadInt64(&nbWorkerToStart) >= int64(maxProv) {
		log.Debug("spawnWorkerForJob> mac concurrent provisioning reached")
		return false, nil
	}

	atomic.AddInt64(&nbWorkerToStart, 1)
	defer func(i *int64) {
		atomic.AddInt64(i, -1)
	}(&nbWorkerToStart)

	if h.Hatchery() == nil || h.Hatchery().ID == 0 {
		log.Debug("spawnWorkerForJob> continue")
		return false, nil
	}

	if len(j.models) == 0 {
		return false, fmt.Errorf("spawnWorkerForJob> %d - No model returned by CDS api", j.timestamp)
	}
	for i := range j.models {
		model := &j.models[i]
		if canRunJob(h, j.timestamp, j.execGroups, j.id, j.requirements, model, j.hostname) {
			if err := h.CDSClient().QueueJobBook(j.isWorkflowJob, j.id); err != nil {
				// perhaps already booked by another hatchery
				log.Debug("spawnWorkerForJob> %d - cannot book job %d %s: %s", j.timestamp, j.id, model.Name, err)
				break // go to next job
			}
			log.Debug("spawnWorkerForJob> %d - send book job %d %s by hatchery %d isWorkflowJob:%t", j.timestamp, j.id, model.Name, h.Hatchery().ID, j.isWorkflowJob)

			start := time.Now()
			infos := []sdk.SpawnInfo{
				{
					RemoteTime: start,
					Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStarts.ID, Args: []interface{}{h.Hatchery().Name, fmt.Sprintf("%d", h.Hatchery().ID), model.Name}},
				},
			}
			workerName, errSpawn := h.SpawnWorker(SpawnArguments{Model: *model, IsWorkflowJob: j.isWorkflowJob, JobID: j.id, Requirements: j.requirements, LogInfo: "spawn for job"})
			if errSpawn != nil {
				log.Warning("spawnWorkerForJob> %d - cannot spawn worker %s for job %d: %s", j.timestamp, model.Name, j.id, errSpawn)
				infos = append(infos, sdk.SpawnInfo{
					RemoteTime: time.Now(),
					Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryErrorSpawn.ID, Args: []interface{}{h.Hatchery().Name, fmt.Sprintf("%d", h.Hatchery().ID), model.Name, sdk.Round(time.Since(start), time.Second).String(), errSpawn.Error()}},
				})
				if err := h.CDSClient().QueueJobSendSpawnInfo(j.isWorkflowJob, j.id, infos); err != nil {
					log.Warning("spawnWorkerForJob> %d - cannot client.QueueJobSendSpawnInfo for job (err spawn)%d: %s", j.timestamp, j.id, err)
				}
				if err := h.CDSClient().WorkerModelSpawnError(model.ID, fmt.Sprintf("hatchery %s cannot spawn worker %s for job %d: %v", h.Hatchery().Name, model.Name, j.id, errSpawn)); err != nil {
					log.Error("spawnWorkerForJob> error on call client.WorkerModelSpawnError on worker model %s for register: %s", model.Name, errSpawn)
				}
				continue // try another model
			}

			infos = append(infos, sdk.SpawnInfo{
				RemoteTime: time.Now(),
				Message: sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStartsSuccessfully.ID,
					Args: []interface{}{
						h.Hatchery().Name,
						fmt.Sprintf("%d", h.Hatchery().ID),
						workerName,
						sdk.Round(time.Since(start), time.Second).String()},
				},
			})

			if err := h.CDSClient().QueueJobSendSpawnInfo(j.isWorkflowJob, j.id, infos); err != nil {
				log.Warning("spawnWorkerForJob> %d - cannot client.QueueJobSendSpawnInfo for job %d: %s", j.timestamp, j.id, err)
			}
			return true, nil // ok for this job
		}
	}

	return false, nil
}

func canRunJob(h Interface, timestamp int64, execGroups []sdk.Group, jobID int64, requirements []sdk.Requirement, model *sdk.Model, hostname string) bool {
	if model.Type != h.ModelType() {
		return false
	}

	// if current hatchery is in same group than worker model -> do not avoid spawn, even if worker model is in error
	if model.NbSpawnErr > 5 && h.Hatchery().GroupID != model.ID {
		log.Warning("canRunJob> Too many errors on spawn with model %s, please check this worker model", model.Name)
		return false
	}

	if len(execGroups) > 0 {
		checkGroup := false
		for _, g := range execGroups {
			if g.ID == model.GroupID {
				checkGroup = true
				break
			}
		}
		if !checkGroup {
			log.Debug("canRunJob> %d - job %d - model %s attached to group %d can't run this job", timestamp, jobID, model.Name, model.GroupID)
			return false
		}
	}

	var containsModelRequirement, containsHostnameRequirement bool
	for _, r := range requirements {
		switch r.Type {
		case sdk.ModelRequirement:
			containsModelRequirement = true
		case sdk.HostnameRequirement:
			containsHostnameRequirement = true
		}
	}
	// Common check
	for _, r := range requirements {
		// If requirement is a Model requirement, it's easy. It's either can or can't run
		// r.Value could be: theModelName --port=8888:9999, so we take strings.Split(r.Value, " ")[0] to compare
		// only modelName
		if r.Type == sdk.ModelRequirement && strings.Split(r.Value, " ")[0] != model.Name {
			log.Debug("canRunJob> %d - job %d - model requirement r.Value(%s) != model.Name(%s)", timestamp, jobID, strings.Split(r.Value, " ")[0], model.Name)
			return false
		}

		// If requirement is an hostname requirement, it's for a specific worker
		if r.Type == sdk.HostnameRequirement && r.Value != hostname {
			log.Debug("canRunJob> %d - job %d - hostname requirement r.Value(%s) != hostname(%s)", timestamp, jobID, r.Value, hostname)
			return false
		}

		// service and memory requirements are only supported by docker model
		if model.Type != sdk.Docker && (r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement) {
			log.Debug("canRunJob> %d - job %d - job with service requirement or memory requirement: only for model docker. current model:%s", timestamp, jobID, model.Type)
			return false
		}

		// Skip network access requirement as we can't check it
		if r.Type == sdk.NetworkAccessRequirement || r.Type == sdk.PluginRequirement || r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			log.Debug("canRunJob> %d - job %d - job with service requirement or memory requirement: only for model docker. current model:%s", timestamp, jobID, model.Type)
			continue
		}

		if r.Type == sdk.OSArchRequirement && model.RegisteredOS != "" && model.RegisteredArch != "" && r.Value != (model.RegisteredOS+"/"+model.RegisteredArch) {
			log.Debug("canRunJob> %d - job %d - job with OSArch requirement: cannot spawn on this OSArch. current model: %s/%s", timestamp, jobID, model.RegisteredOS, model.RegisteredArch)
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
					log.Debug("canRunJob> %d - job %d - model(%s) does not have binary %s(%s) for this job.", timestamp, jobID, model.Name, r.Name, r.Value)
					return false
				}
			}
		}
	}

	// If the model needs registration, don't spawn for now
	if h.NeedRegistration(model) {
		log.Debug("canRunJob> model %s needs registration", model.Name)
		return false
	}

	return h.CanSpawn(model, jobID, requirements)
}
