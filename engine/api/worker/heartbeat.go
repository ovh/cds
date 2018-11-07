package worker

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const workerHeartbeatTimeout = 300.0

// disableDeadWorkers put status disabled to all dead workers with status Registering, Waiting or Building
func disableDeadWorkers(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	workers, err := LoadDeadWorkers(db, workerHeartbeatTimeout, []string{sdk.StatusWorkerRegistering.String(), sdk.StatusBuilding.String(), sdk.StatusWaiting.String()})
	if err != nil {
		return sdk.WrapError(err, "Cannot load dead workers")
	}
	for i := range workers {
		log.Debug("Disable worker %s[%s] LastBeat:%v hatchery:%s status:%s", workers[i].Name, workers[i].ID, workers[i].LastBeat, workers[i].HatcheryName, workers[i].Status)
		if errD := SetStatus(db, workers[i].ID, sdk.StatusDisabled); errD != nil {
			log.Warning("Cannot disable worker %v: %v", workers[i].ID, errD)
		}
	}

	return nil
}

// deleteDeadWorkers delete all workers which is disabled
func deleteDeadWorkers(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	workers, err := LoadDeadWorkers(db, workerHeartbeatTimeout, []string{sdk.StatusDisabled.String()})
	if err != nil {
		return sdk.WrapError(err, "Cannot load dead workers")
	}
	for i := range workers {
		log.Debug("deleteDeadWorkers> Delete worker %s[%s] LastBeat:%v hatchery:%s status:%s", workers[i].Name, workers[i].ID, workers[i].LastBeat, workers[i].HatcheryName, workers[i].Status)
		tx, err := db.Begin()
		if err != nil {
			log.Error("deleteDeadWorkers> Cannot create transaction")
		}

		if errD := DeleteWorker(tx, workers[i].ID); errD != nil {
			log.Warning("deleteDeadWorkers> Cannot delete worker %v: %v", workers[i].ID, errD)
			_ = tx.Rollback()
			continue
		}

		if _, errU := tx.Exec("UPDATE workflow_node_run_job SET worker_id = NULL WHERE worker_id = $1", workers[i].ID); errU != nil {
			log.Warning("deleteDeadWorkers> Cannot update workflow_node_run_job : %v", errU)
			_ = tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Error("deleteDeadWorkers> Cannot commit transaction : %v", err)
		}
	}

	return nil
}
