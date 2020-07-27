package workflow

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const maxRetry = 3

// manageDeadJob restart all jobs which are building but without worker
func manageDeadJob(ctx context.Context, DBFunc func() *gorp.DbMap, store cache.Store, maxLogSize int64) error {
	db := DBFunc()
	deadJobs, err := LoadDeadNodeJobRun(ctx, db, store)
	if err != nil {
		return sdk.WrapError(err, "Cannot load dead node job run")
	}

	for _, deadJob := range deadJobs {
		tx, errTx := db.Begin()
		if errTx != nil {
			log.Error(ctx, "manageDeadJob> Cannot create transaction : %v", errTx)
			continue
		}

		if deadJob.Status == sdk.StatusBuilding {
			if deadJob.Retry >= maxRetry {
				if _, err := UpdateNodeJobRunStatus(ctx, tx, store, sdk.Project{}, &deadJob, sdk.StatusStopped); err != nil {
					log.Error(ctx, "manageDeadJob> Cannot update node run job %d : %v", deadJob.ID, err)
					_ = tx.Rollback()
					continue
				}

				if err := DeleteNodeJobRun(tx, deadJob.ID); err != nil {
					log.Error(ctx, "manageDeadJob> Cannot delete node run job %d : %v", deadJob.ID, err)
					_ = tx.Rollback()
					continue
				}
			} else {
				if err := RestartWorkflowNodeJob(ctx, tx, deadJob, maxLogSize); err != nil {
					log.Warning(ctx, "manageDeadJob> Cannot restart node job run %d: %v", deadJob.ID, err)
					_ = tx.Rollback()
					continue
				}
			}
		} else if sdk.StatusIsTerminated(deadJob.Status) {
			if err := DeleteNodeJobRun(tx, deadJob.ID); err != nil {
				log.Error(ctx, "manageDeadJob> Cannot delete node run job %d : %v", deadJob.ID, err)
				_ = tx.Rollback()
				continue
			}
		}

		if err := tx.Commit(); err != nil {
			log.Error(ctx, "manageDeadJob> Cannot commit transaction : %v", err)
		}
	}

	return nil
}
