package workflow

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// manageDeadJob restart all jobs which are building but without worker
func manageDeadJob(ctx context.Context, DBFunc func() *gorp.DbMap, store cache.Store, maxLogSize int64) error {
	db := DBFunc()
	deadJobs, err := LoadDeadNodeJobRun(ctx, db, store)
	if err != nil {
		return sdk.WrapError(err, "Cannot load dead node job run")
	}

	for _, deadJob := range deadJobs {
		tx, err := db.Begin()
		if err != nil {
			log.Error(ctx, "manageDeadJob> Cannot create transaction : %v", err)
			continue
		}

		if deadJob.Status == sdk.StatusBuilding {
			log.Info(ctx, "manageDeadJob> set job %v to fail", deadJob.ID)
			if _, err := UpdateNodeJobRunStatus(ctx, tx, store, sdk.Project{}, &deadJob, sdk.StatusFail); err != nil {
				log.Error(ctx, "manageDeadJob> Cannot update node run job %d : %v", deadJob.ID, err)
				_ = tx.Rollback()
				continue
			}
			if err := DeleteNodeJobRun(tx, deadJob.ID); err != nil {
				log.Error(ctx, "manageDeadJob> Cannot delete node run job %d : %v", deadJob.ID, err)
				_ = tx.Rollback()
				continue
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
