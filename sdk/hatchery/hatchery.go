package hatchery

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Interface describe an interface for each hatchery mode (mesos, local)
type Interface interface {
	Init() error
	KillWorker(worker sdk.Worker) error
	SpawnWorker(model *sdk.Model, job *sdk.PipelineBuildJob) error
	CanSpawn(model *sdk.Model, job *sdk.PipelineBuildJob) bool
	WorkersStartedByModel(model *sdk.Model) int
	WorkersStarted() int
	Hatchery() *sdk.Hatchery
	ModelType() string
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

func routine(h Interface, maxWorkers, provision int, hostname string, timestamp int64, lastSpawnedIDs []int64, warningSeconds, criticalSeconds, graceSeconds int) ([]int64, error) {
	defer logTime(fmt.Sprintf("routine> %d", timestamp), time.Now(), warningSeconds, criticalSeconds)
	log.Debug("routine> %d enter", timestamp)

	if h.Hatchery() == nil || h.Hatchery().ID == 0 {
		log.Debug("Create> continue")
		return nil, nil
	}

	workersStarted := h.WorkersStarted()
	if workersStarted > maxWorkers {
		log.Notice("routine> %d max workers reached. current:%d max:%d", timestamp, workersStarted, maxWorkers)
		return nil, nil
	}
	log.Debug("routine> %d - workers already started:%d", timestamp, workersStarted)

	jobs, errbq := sdk.GetBuildQueue()
	if errbq != nil {
		log.Critical("routine> %d error on GetBuildQueue:%e", timestamp, errbq)
		return nil, errbq
	}

	if len(jobs) == 0 {
		log.Debug("routine> %d - Job queue is empty", timestamp)
		return nil, nil
	}
	log.Debug("routine> %d - Job queue size:%d", timestamp, len(jobs))

	models, errwm := sdk.GetWorkerModels()
	if errwm != nil {
		log.Debug("routine> %d - error on GetWorkerModels:%e", timestamp, errwm)
		return nil, errwm
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("routine> %d - No model returned by GetWorkerModels", timestamp)
	}
	log.Debug("routine> %d - models received: %d", timestamp, len(models))

	spawnedIDs := []int64{}
	wg := &sync.WaitGroup{}

	nToRun := len(jobs)
	if len(jobs) > maxWorkers-workersStarted {
		nToRun = maxWorkers - workersStarted
		if nToRun < 0 { // should never occur, just to be sure
			nToRun = 1
		}
		log.Info("routine> %d - work only on %d jobs from queue. queue size:%d workersStarted:%d maxWorkers:%d", timestamp, nToRun, len(jobs), workersStarted, maxWorkers)
	}

	for i := range jobs[:nToRun] {
		wg.Add(1)
		go func(job *sdk.PipelineBuildJob) {
			defer logTime(fmt.Sprintf("routine> %d - job %d>", timestamp, job.ID), time.Now(), warningSeconds, criticalSeconds)

			if sdk.IsInArray(job.ID, lastSpawnedIDs) {
				log.Debug("routine> %d - job %d already spawned in previous routine", timestamp, job.ID)
				wg.Done()
				return
			}

			if job.QueuedSeconds < int64(graceSeconds) {
				log.Debug("routine> %d - job %d is too fresh, queued since %d seconds, let existing waiting worker check it", timestamp, job.ID, job.QueuedSeconds)
				wg.Done()
				return
			}

			log.Debug("routine> %d - work on job %d queued since %d seconds", timestamp, job.ID, job.QueuedSeconds)
			if job.BookedBy.ID != 0 {
				t := "current hatchery"
				if job.BookedBy.ID != h.Hatchery().ID {
					t = "another hatchery"
				}
				log.Debug("routine> %d - job %d already booked by %s %s (%d)", timestamp, job.ID, t, job.BookedBy.Name, job.BookedBy.ID)
				wg.Done()
				return
			}

			for _, model := range models {
				if canRunJob(h, timestamp, job, &model, hostname) {
					if err := sdk.BookPipelineBuildJob(job.ID); err != nil {
						// perhaps already booked by another hatchery
						log.Debug("routine> %d - cannot book job %d %s: %s", timestamp, job.ID, model.Name, err)
						break // go to next job
					}
					log.Debug("routine> %d - send book job %d %s by hatchery %d", timestamp, job.ID, model.Name, h.Hatchery().ID)

					start := time.Now()
					infos := []sdk.SpawnInfo{
						{
							RemoteTime: start,
							Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStarts.ID, Args: []interface{}{fmt.Sprintf("%d", h.Hatchery().ID), model.Name}},
						},
					}

					if err := h.SpawnWorker(&model, job); err != nil {
						log.Warning("routine> %d - cannot spawn worker %s for job %d: %s", timestamp, model.Name, job.ID, err)
						infos = append(infos, sdk.SpawnInfo{
							RemoteTime: time.Now(),
							Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryErrorSpawn.ID, Args: []interface{}{fmt.Sprintf("%d", h.Hatchery().ID), model.Name, sdk.Round(time.Since(start), time.Second).String(), err.Error()}},
						})
						if err := sdk.AddSpawnInfosPipelineBuildJob(job.ID, infos); err != nil {
							log.Warning("routine> %d - cannot record AddSpawnInfosPipelineBuildJob for job (err spawn)%d: %s", timestamp, job.ID, err)
						}
						continue // try another model
					}
					spawnedIDs = append(spawnedIDs, job.ID)

					infos = append(infos, sdk.SpawnInfo{
						RemoteTime: time.Now(),
						Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStartsSuccessfully.ID, Args: []interface{}{fmt.Sprintf("%d", h.Hatchery().ID), sdk.Round(time.Since(start), time.Second).String()}},
					})

					if err := sdk.AddSpawnInfosPipelineBuildJob(job.ID, infos); err != nil {
						log.Warning("routine> %d - cannot record AddSpawnInfosPipelineBuildJob for job %d: %s", timestamp, job.ID, err)
					}
					break // ok for this job
				}
			}
			wg.Done()
		}(&jobs[i])
	}

	wg.Wait()

	return spawnedIDs, nil
}

func provisioning(h Interface, provision int) {
	if provision == 0 {
		log.Debug("provisioning> no provisioning to do")
		return
	}

	models, errwm := sdk.GetWorkerModels()
	if errwm != nil {
		log.Debug("provisioning> error on GetWorkerModels:%e", errwm)
		return
	}

	for k := range models {
		if h.WorkersStartedByModel(&models[k]) < provision {
			if models[k].Type == h.ModelType() {
				go func(m sdk.Model) {
					if err := h.SpawnWorker(&m, nil); err != nil {
						log.Warning("provisioning> cannot spawn worker for provisioning: %s", m.Name, err)
					}
				}(models[k])
			}
		}
	}
}

func canRunJob(h Interface, timestamp int64, job *sdk.PipelineBuildJob, model *sdk.Model, hostname string) bool {
	if model.Type != h.ModelType() {
		return false
	}

	// Common check
	for _, r := range job.Job.Action.Requirements {
		// If requirement is a Model requirement, it's easy. It's either can or can't run
		if r.Type == sdk.ModelRequirement && r.Value != model.Name {
			log.Debug("canRunJob> %d - job %d - model requirement r.Value(%s) != model.Name(%s)", timestamp, job.ID, r.Value, model.Name)
			return false
		}

		// If requirement is an hostname requirement, it's for a specific worker
		if r.Type == sdk.HostnameRequirement && r.Value != hostname {
			log.Debug("canRunJob> %d - job %d - hostname requirement r.Value(%s) != hostname(%s)", timestamp, job.ID, r.Value, hostname)
			return false
		}

		// service and memory requirements are only supported by docker model
		if model.Type != sdk.Docker && (r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement) {
			log.Debug("canRunJob> %d - job %d - job with service requirement or memory requirement: only for model docker. current model:%s", timestamp, job.ID, model.Type)
			return false
		}

		// Skip network access requirement as we can't check it
		if r.Type == sdk.NetworkAccessRequirement || r.Type == sdk.PluginRequirement || r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			log.Debug("canRunJob> %d - job %d - job with service requirement or memory requirement: only for model docker. current model:%s", timestamp, job.ID, model.Type)
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
				log.Debug("canRunJob> %d - job %d - model(%s) does not have binary %s(%s) for this job.", timestamp, job.ID, model.Name, r.Name, r.Value)
				return false
			}
		}
	}

	return h.CanSpawn(model, job)
}

func logTime(name string, then time.Time, warningSeconds, criticalSeconds int) {
	d := time.Since(then)
	if d > time.Duration(criticalSeconds)*time.Second {
		log.Critical("%s took %s to execute", name, d)
		return
	}

	if d > time.Duration(warningSeconds)*time.Second {
		log.Warning("%s took %s to execute", name, d)
		return
	}

	log.Debug("%s took %s to execute", name, d)
}
