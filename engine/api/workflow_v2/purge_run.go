package workflow_v2

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/rockbears/log"
)

func PurgeWorkflowRun(ctx context.Context, DBFunc func() *gorp.DbMap, purgeRoutineTIcker int64) {
	tickPurge := time.NewTicker(time.Duration(purgeRoutineTIcker) * time.Minute)
	defer tickPurge.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting purge workflow: %v", ctx.Err())
				return
			}
		case <-tickPurge.C:
			ids, err := LoadRunIDsToDelete(ctx, DBFunc())
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
			}
			for _, id := range ids {
				if err := DeleteRun(ctx, DBFunc(), id); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			}
		}
	}

}

func DeleteRun(ctx context.Context, db *gorp.DbMap, id string) error {
	ctx = context.WithValue(ctx, cdslog.WorkflowRunID, id)

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	run, err := LoadAndLockRunByID(ctx, db, id)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil
		}
		return err
	}
	ctx = context.WithValue(ctx, cdslog.Project, run.ProjectKey)
	ctx = context.WithValue(ctx, cdslog.Workflow, run.WorkflowName)

	dbRun := dbWorkflowRun{V2WorkflowRun: *run}
	if err := gorpmapping.Delete(db, &dbRun); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	log.Info(ctx, "run %s / %s / %s deleted", run.ProjectKey, run.WorkflowName, run.ID)
	return nil
}
