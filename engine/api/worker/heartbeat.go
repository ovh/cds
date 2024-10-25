package worker

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

const workerHeartbeatTimeout = 300.0

// DisableDeadWorkers put status disabled to all dead workers with status Registering, Waiting or Building
func DisableDeadWorkers(ctx context.Context, db *gorp.DbMap) error {
	workers, err := LoadDeadWorkers(ctx, db, workerHeartbeatTimeout, []string{sdk.StatusWorkerRegistering, sdk.StatusBuilding, sdk.StatusWaiting})
	if err != nil {
		return sdk.WrapError(err, "Cannot load dead workers")
	}
	for i := range workers {
		tx, err := db.Begin()
		if err != nil {
			log.Error(ctx, "DisableDeadWorkers> unable to start transaction: %v", err)
			continue
		}

		log.Debug(ctx, "Disable worker %s[%s] LastBeat:%v status:%s", workers[i].Name, workers[i].ID, workers[i].LastBeat, workers[i].Status)
		if err := SetStatus(ctx, tx, workers[i].ID, sdk.StatusDisabled); err != nil {
			_ = tx.Rollback()
			log.Warn(ctx, "Cannot disable worker %v: %v", workers[i].ID, err)
			continue
		}

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			log.Error(ctx, "DisableDeadWorkers> unable to commit transaction: %v", err)
			continue
		}
	}

	return nil
}

// DeleteDeadWorkers delete all workers which is disabled
func DeleteDeadWorkers(ctx context.Context, db *gorp.DbMap) error {
	workers, err := LoadDeadWorkers(ctx, db, workerHeartbeatTimeout, []string{sdk.StatusDisabled})
	if err != nil {
		return sdk.WrapError(err, "Cannot load dead workers")
	}
	for i := range workers {
		log.Debug(ctx, "deleteDeadWorkers> Delete worker %s[%s] LastBeat:%v status:%s", workers[i].Name, workers[i].ID, workers[i].LastBeat, workers[i].Status)
		tx, err := db.Begin()
		if err != nil {
			log.Error(ctx, "deleteDeadWorkers> Cannot create transaction")
			continue
		}

		if errD := Delete(tx, workers[i].ID); errD != nil {
			log.Warn(ctx, "deleteDeadWorkers> Cannot delete worker %v: %v", workers[i].ID, errD)
			_ = tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Error(ctx, "deleteDeadWorkers> Cannot commit transaction : %v", err)
		}
	}

	return nil
}
