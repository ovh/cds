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
			nodeRun, err := LoadAndLockNodeRunByID(ctx, tx, deadJob.WorkflowNodeRunID)
			if err != nil {
				return sdk.WrapError(err, "manageDeadJob> cannot load node run: %d", deadJob.WorkflowNodeRunID)
			}
			infos := []sdk.SpawnInfo{{
				Message: sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobFailedCauseByWorkerLost.ID, Args: []interface{}{deadJob.ID}},
			}}
			if err := AddSpawnInfosNodeJobRun(tx, deadJob.WorkflowNodeRunID, deadJob.ID, infos); err != nil {
				return sdk.WrapError(err, "cannot save spawn info on node job run %d", deadJob.ID)
			}
			sync, err := SyncNodeRunRunJob(ctx, tx, nodeRun, deadJob)
			if err != nil {
				return sdk.WrapError(err, "manageDeadJob> unable to sync nodeJobRun. JobID on handler: %d", deadJob.ID)
			}
			if !sync {
				log.Warn(ctx, "manageDeadJob> sync doesn't find a nodeJobRun. JobID on handler: %d", deadJob.ID)
			}
			if err := UpdateNodeRun(tx, nodeRun); err != nil {
				return sdk.WrapError(err, "manageDeadJob> cannot update node run. JobID on handler: %d", deadJob.ID)
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
