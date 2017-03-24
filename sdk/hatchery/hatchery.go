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
	WorkerStarted(model *sdk.Model) int
	Hatchery() *sdk.Hatchery
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

func routine(h Interface, provision int, hostname string, timestamp int64, lastSpawnedIDs []int64, warningSeconds, criticalSeconds, graceSeconds int) ([]int64, error) {
	defer logTime("routine", time.Now(), warningSeconds, criticalSeconds)
	log.Debug("routine> %d", timestamp)

	jobs, errbq := sdk.GetBuildQueue()
	if errbq != nil {
		log.Critical("routine> %d error on GetBuildQueue:%e", timestamp, errbq)
		return nil, errbq
	}

	if len(jobs) == 0 {
		log.Debug("routine> %d - Job queue is empty", timestamp)
		return nil, nil
	}

	models, errwm := sdk.GetWorkerModels()
	if errwm != nil {
		log.Debug("routine> %d - error on GetWorkerModels:%e", timestamp, errwm)
		return nil, errwm
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("routine> %d - No model returned by GetWorkerModels", timestamp)
	}
	log.Debug("routine> %d models received", len(models))

	spawnedIDs := []int64{}
	wg := &sync.WaitGroup{}

	for i := range jobs {
		wg.Add(1)
		go func(job *sdk.PipelineBuildJob) {
			defer logTime(fmt.Sprintf("routine> job %d>", job.ID), time.Now(), warningSeconds, criticalSeconds)

			if sdk.IsInArray(job.ID, lastSpawnedIDs) {
				log.Debug("routine> job %d already spawned in previous routine", job.ID)
				wg.Done()
				return
			}

			if job.QueuedSeconds < int64(graceSeconds) {
				log.Debug("routine> job %d is too fresh, queued since %d seconds, let existing waiting worker check it", job.ID, job.QueuedSeconds)
				wg.Done()
				return
			}

			log.Debug("routine> %d work on job %d queued since %d seconds", timestamp, job.ID, job.QueuedSeconds)
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
				if canRunJob(h, job, &model, hostname) {
					if err := sdk.BookPipelineBuildJob(job.ID); err != nil {
						// perhaps already booked by another hatchery
						log.Debug("routine> %d cannot book job %d %s: %s", timestamp, job.ID, model.Name, err)
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

func canRunJob(h Interface, job *sdk.PipelineBuildJob, model *sdk.Model, hostname string) bool {
	if !h.CanSpawn(model, job) {
		return false
	}

	// Common check
	for _, r := range job.Job.Action.Requirements {
		// If requirement is a Model requirement, it's easy. It's either can or can't run
		if r.Type == sdk.ModelRequirement {
			return r.Value == model.Name
		}

		// If requirement is an hostname requirement, it's for a specific worker
		if r.Type == sdk.HostnameRequirement {
			return r.Value == hostname
		}

		// service and memory requirements are only supported by docker model
		if model.Type != sdk.Docker && (r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement) {
			return false
		}

		found := false

		// Skip network access requirement as we can't check it
		if r.Type == sdk.NetworkAccessRequirement || r.Type == sdk.PluginRequirement || r.Type == sdk.ServiceRequirement || r.Type == sdk.MemoryRequirement {
			continue
		}

		// Check binary requirement against worker model capabilities
		for _, c := range model.Capabilities {
			if r.Value == c.Value || r.Value == c.Name {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
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
