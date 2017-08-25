package hatchery

import (
	"fmt"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// Interface describe an interface for each hatchery mode (mesos, local)
type Interface interface {
	Init(name, api, token string, requestSecondsTimeout int, insecureSkipVerifyTLS bool) error
	KillWorker(worker sdk.Worker) error
	SpawnWorker(model *sdk.Model, jobID int64, requirements []sdk.Requirement, registerOnly bool, logInfo string) (string, error)
	CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool
	WorkersStartedByModel(model *sdk.Model) int
	WorkersStarted() int
	Hatchery() *sdk.Hatchery
	Client() cdsclient.Interface
	ModelType() string
	NeedRegistration(model *sdk.Model) bool
	ID() int64
}

var (
	// Client is a CDS Client
	Client sdk.HTTPClient
)

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

func receiveJob(h Interface, jobID int64, jobQueuedSeconds int64, jobBookedBy sdk.Hatchery, requirements []sdk.Requirement, models []sdk.Model, nRoutines *int64, spawnIDs *cache.Cache, warningSeconds, criticalSeconds, graceSeconds int, hostname string) bool {
	if jobID == 0 {
		return false
	}

	n := atomic.LoadInt64(nRoutines)
	if n > 10 {
		log.Info("too many routines in same time %d", n)
		return false
	}

	if _, exist := spawnIDs.Get(string(jobID)); exist {
		log.Debug("job %d already spawned in previous routine", jobID)
		return false
	}

	if jobQueuedSeconds < int64(graceSeconds) {
		log.Debug("job %d is too fresh, queued since %d seconds, let existing waiting worker check it", jobID, jobQueuedSeconds)
		return false
	}

	log.Debug("work on job %d queued since %d seconds", jobID, jobQueuedSeconds)
	if jobBookedBy.ID != 0 {
		t := "current hatchery"
		if jobBookedBy.ID != h.Hatchery().ID {
			t = "another hatchery"
		}
		log.Debug("job %d already booked by %s %s (%d)", jobID, t, jobBookedBy.Name, jobBookedBy.ID)
		return false
	}

	atomic.AddInt64(nRoutines, 1)
	defer atomic.AddInt64(nRoutines, -1)
	if errR := routine(h, models, jobID, requirements, hostname, time.Now().Unix(), warningSeconds, criticalSeconds, graceSeconds); errR != nil {
		log.Warning("Error on routine: %s", errR)
		return false
	}
	return true
}

func routine(h Interface, models []sdk.Model, jobID int64, requirements []sdk.Requirement, hostname string, timestamp int64, warningSeconds, criticalSeconds, graceSeconds int) error {
	defer logTime(fmt.Sprintf("routine> %d", timestamp), time.Now(), warningSeconds, criticalSeconds)
	log.Debug("routine> %d enter", timestamp)

	if h.Hatchery() == nil || h.Hatchery().ID == 0 {
		log.Debug("Create> continue")
		return nil
	}

	if len(models) == 0 {
		return fmt.Errorf("routine> %d - No model returned by CDS api", timestamp)
	}
	log.Debug("routine> %d - models received: %d", timestamp, len(models))

	for _, model := range models {
		if canRunJob(h, timestamp, jobID, requirements, &model, hostname) {
			if err := sdk.BookPipelineBuildJob(jobID); err != nil {
				// perhaps already booked by another hatchery
				log.Debug("routine> %d - cannot book job %d %s: %s", timestamp, jobID, model.Name, err)
				break // go to next job
			}
			log.Debug("routine> %d - send book job %d %s by hatchery %d", timestamp, jobID, model.Name, h.Hatchery().ID)

			start := time.Now()
			infos := []sdk.SpawnInfo{
				{
					RemoteTime: start,
					Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStarts.ID, Args: []interface{}{fmt.Sprintf("%s", h.Hatchery().Name), fmt.Sprintf("%d", h.Hatchery().ID), model.Name}},
				},
			}
			workerName, errSpawn := h.SpawnWorker(&model, jobID, requirements, false, "spawn for job")
			if errSpawn != nil {
				log.Warning("routine> %d - cannot spawn worker %s for job %d: %s", timestamp, model.Name, jobID, errSpawn)
				infos = append(infos, sdk.SpawnInfo{
					RemoteTime: time.Now(),
					Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryErrorSpawn.ID, Args: []interface{}{fmt.Sprintf("%s", h.Hatchery().Name), fmt.Sprintf("%d", h.Hatchery().ID), model.Name, sdk.Round(time.Since(start), time.Second).String(), errSpawn.Error()}},
				})
				if err := sdk.AddSpawnInfosPipelineBuildJob(jobID, infos); err != nil {
					log.Warning("routine> %d - cannot record AddSpawnInfosPipelineBuildJob for job (err spawn)%d: %s", timestamp, jobID, err)
				}
				if err := sdk.SpawnErrorWorkerModel(model.ID, fmt.Sprintf("routine> cannot spawn worker %s for job %d: %s", model.Name, jobID, errSpawn)); err != nil {
					log.Error("routine> error on call sdk.SpawnErrorWorkerModel on worker model %s for register: %s", model.Name, errSpawn)
				}
				continue // try another model
			}

			infos = append(infos, sdk.SpawnInfo{
				RemoteTime: time.Now(),
				Message: sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStartsSuccessfully.ID,
					Args: []interface{}{
						fmt.Sprintf("%s", h.Hatchery().Name),
						fmt.Sprintf("%d", h.Hatchery().ID),
						fmt.Sprintf("%s", workerName),
						sdk.Round(time.Since(start), time.Second).String()},
				},
			})

			if err := sdk.AddSpawnInfosPipelineBuildJob(jobID, infos); err != nil {
				log.Warning("routine> %d - cannot record AddSpawnInfosPipelineBuildJob for job %d: %s", timestamp, jobID, err)
			}
			break // ok for this job
		}
	}

	return nil
}

func provisioning(h Interface, provisionDisabled bool, models []sdk.Model) {
	if provisionDisabled {
		log.Debug("provisioning> disabled on this hatchery")
		return
	}

	for k := range models {
		if models[k].Type == h.ModelType() {
			existing := h.WorkersStartedByModel(&models[k])
			for i := existing; i < int(models[k].Provision); i++ {
				go func(m sdk.Model) {
					if name, errSpawn := h.SpawnWorker(&m, 0, nil, false, "spawn for provision"); errSpawn != nil {
						log.Warning("provisioning> cannot spawn worker %s with model %s for provisioning: %s", name, m.Name, errSpawn)
						if err := sdk.SpawnErrorWorkerModel(m.ID, fmt.Sprintf("routine> cannot spawn worker %s for provisioning: %s", m.Name, errSpawn)); err != nil {
							log.Error("provisioning> cannot spawn worker %s with model %s for provisioning: %s", name, m.Name, errSpawn)
						}
					}
				}(models[k])
			}
		}
	}
}

func canRunJob(h Interface, timestamp int64, jobID int64, requirements []sdk.Requirement, model *sdk.Model, hostname string) bool {
	if model.Type != h.ModelType() {
		return false
	}

	// if current hatchery is in same group than worker model -> do not avoid spawn, even if worker model is in error
	if model.NbSpawnErr > 5 && h.Hatchery().GroupID != model.ID {
		log.Warning("canRunJob> Too many errors on spawn with model %s, please check this worker model", model.Name)
		return false
	}

	// Common check
	for _, r := range requirements {
		// If requirement is a Model requirement, it's easy. It's either can or can't run
		if r.Type == sdk.ModelRequirement && r.Value != model.Name {
			log.Debug("canRunJob> %d - job %d - model requirement r.Value(%s) != model.Name(%s)", timestamp, jobID, r.Value, model.Name)
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

		if r.Type == sdk.BinaryRequirement {
			found := false
			// Check binary requirement against worker model capabilities
			for _, c := range model.Capabilities {
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

	return h.CanSpawn(model, jobID, requirements)
}

func logTime(name string, then time.Time, warningSeconds, criticalSeconds int) {
	d := time.Since(then)
	if d > time.Duration(criticalSeconds)*time.Second {
		log.Error("%s took %s to execute", name, d)
		return
	}

	if d > time.Duration(warningSeconds)*time.Second {
		log.Warning("%s took %s to execute", name, d)
		return
	}

	log.Debug("%s took %s to execute", name, d)
}
