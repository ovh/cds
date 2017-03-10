package hatchery

import (
	"os/exec"

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

func routine(h Interface, provision int, hostname string) error {
	log.Debug("routine> ")

	jobs, errbq := sdk.GetBuildQueue()
	if errbq != nil {
		log.Debug("routine> err while GetBuildQueue:%e\n", errbq)
		return errbq
	}

	if len(jobs) == 0 {
		log.Debug("routine> Job queue is empty")
		return nil
	}

	models, errwm := sdk.GetWorkerModels()
	if errwm != nil {
		log.Debug("routine> err while GetWorkerModels:%e\n", errwm)
		return errwm
	}

	for _, job := range jobs {
		if job.BookedBy.ID != 0 {
			t := "current hatchery"
			if job.BookedBy.ID != h.Hatchery().ID {
				t = "another hatchery"
			}
			log.Debug("routine> job %d already booked by %s %s (%d)\n", job.ID, t, job.BookedBy.Name, job.BookedBy.ID)
			continue
		}

		for _, model := range models {
			if canRunJob(h, &job, &model, hostname) {
				if err := sdk.BookPipelineBuildJob(job.ID); err != nil {
					log.Notice("routine> cannot book job %d %s: %s\n", job.ID, model.Name, err)
					break
				}
				log.Debug("routine> send book job %d %s by h:%d\n", job.ID, model.Name, h.Hatchery().ID)

				if err := h.SpawnWorker(&model, &job); err != nil {
					log.Warning("routine> cannot spawn worker %s for job %d: %s\n", model.Name, job.ID, err)
					break
				}
			}
		}
	}

	return nil
}

func canRunJob(h Interface, job *sdk.PipelineBuildJob, model *sdk.Model, hostname string) bool {

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

	return h.CanSpawn(model, job)
}
