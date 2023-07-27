package worker_v2

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func DeleteDisabledWorker(ctx context.Context, DBFunc func() *gorp.DbMap) {
	db := DBFunc()
	tickDelete := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting worker ticker: %v", ctx.Err())
			}
		case <-tickDelete.C:
			if err := deleteDisabledWorker(ctx, db); err != nil {
				log.Warn(ctx, "worker.DeleteDisabledWorker> Error on deleteDisabledWorker : %v", err)
			}
		}
	}
}

func deleteDisabledWorker(ctx context.Context, db *gorp.DbMap) error {
	workers, err := loadWorkerByStatus(ctx, db, sdk.StatusDisabled)
	if err != nil {
		return sdk.WrapError(err, "Cannot load dead workers")
	}
	for i := range workers {
		tx, err := db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}

		if err := deleteWorker(ctx, tx, workers[i]); err != nil {
			log.Warn(ctx, "deleteWorker> Cannot delete worker %v: %v", workers[i].ID, err)
			_ = tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}
	}
	return nil
}
