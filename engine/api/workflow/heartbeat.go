package workflow

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const maxRetry = 3

// restartDeadJob restart all jobs which are building but without worker
func restartDeadJob(ctx context.Context, DBFunc func() *gorp.DbMap, store cache.Store) error {
	db := DBFunc()
	deadJobs, err := LoadDeadNodeJobRun(db, store)
	if err != nil {
		return sdk.WrapError(err, "Cannot load dead node job run")
	}

	for _, deadJob := range deadJobs {
		tx, errTx := db.Begin()
		if errTx != nil {
			log.Error("restartDeadJob> Cannot create transaction : %v", errTx)
			continue
		}

		if deadJob.Retry >= maxRetry {
			if _, err := UpdateNodeJobRunStatus(ctx, DBFunc, tx, store, nil, &deadJob, sdk.StatusStopped); err != nil {
				log.Error("restartDeadJob> Cannot update node run job %d : %v", deadJob.ID, err)
				_ = tx.Rollback()
				continue
			}

			if err := DeleteNodeJobRuns(tx, deadJob.WorkflowNodeRunID); err != nil {
				log.Error("restartDeadJob> Cannot delete node run job %d : %v", deadJob.ID, err)
				_ = tx.Rollback()
				continue
			}
		} else {
			if err := RestartWorkflowNodeJob(ctx, tx, deadJob); err != nil {
				log.Warning("restartDeadJob> Cannot restart node job run %d: %v", deadJob.ID, err)
				_ = tx.Rollback()
				continue
			}
		}

		if err := tx.Commit(); err != nil {
			log.Error("restartDeadJob> Cannot commit transaction : %v", err)
		}
	}

	return nil
}
