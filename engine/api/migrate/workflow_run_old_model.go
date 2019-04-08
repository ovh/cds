package migrate

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// WorkflowRunOldModel migrates workflow run with old workflow struct to the new one
func WorkflowRunOldModel(ctx context.Context, DBFunc func() *gorp.DbMap) error {
	db := DBFunc()

	log.Info("migrate>WorkflowRunOldModel> Start migration")

	for {
		ids, err := workflow.LoadRunIDsWithOldModel(db)
		if err != nil {
			return err
		}

		if len(ids) == 0 {
			break
		}

		log.Info("migrate>WorkflowRunOldModel> %d run to migrate", len(ids))
		for _, id := range ids {
			if err := migrateRun(ctx, db, id); err != nil && !sdk.ErrorIs(err, sdk.ErrLocked) {
				return err
			}
		}
	}

	log.Info("End WorkflowRunOldModel migration")
	return nil
}

func migrateRun(ctx context.Context, db *gorp.DbMap, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	run, err := workflow.LockRun(tx, id)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrLocked) {
			// Already lock, go to next run
			return nil
		}
		return err
	}

	if err := workflow.MigrateWorkflowRun(ctx, tx, run); err != nil {
		return err
	}

	return sdk.WithStack(tx.Commit())
}
