package hatchery

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// killWorker gets all workers spawned by current hatchery
// and kill all workers with status Waiting
func killWorker(h Interface, model *sdk.Model) error {

	workers, errW := sdk.GetWorkers()
	if errW != nil {
		return errW
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
		if worker.Status != sdk.StatusBuilding {
			if err := sdk.DisableWorker(worker.ID); err != nil {
				return err
			}
			log.Info("KillWorker> Disabled %s\n", worker.Name)
			return h.KillWorker(worker)
		}
		log.Info("KillWorker> Cannot kill building worker %s\n", worker.Name)
	}

	return nil
}
