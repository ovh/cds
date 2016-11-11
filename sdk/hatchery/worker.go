package hatchery

import (
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// killWorker gets all workers spawned by current hatchery
// and kill all workers with status Waiting
func killWorker(h Interface, model *sdk.Model) error {

	workers, err := sdk.GetWorkers()
	if err != nil {
		return err
	}

	// Get list of workers for this model
	for _, worker := range workers {
		if worker.Model != model.ID {
			continue
		}

		// Check if worker was spawned by this hatchery
		if worker.HatcheryID == 0 || worker.HatcheryID != h.ID() {
			continue
		}

		// If worker is not currently executing an action
		if worker.Status == sdk.StatusWaiting {
			// then disable him
			if err = sdk.DisableWorker(worker.ID); err != nil {
				return err
			}
			log.Notice("KillWorker> Disabled %s\n", worker.Name)
			return h.KillWorker(worker)
		}
	}

	return nil
}
