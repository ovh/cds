package migrate

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// WorkflowRunOldModel migrates workflow run with old workflow struct to the new one
func WorkflowRunOldModel(ctx context.Context, DBFunc func() *gorp.DbMap) {
	db := DBFunc()

	log.Info("migrate>WorkflowRunOldModel> Start migration")

	var ids []int64
	ids, err := workflow.LoadRunIDsWithOldModel(db)
	if err != nil {
		log.Error("migrate>WorkflowRunOldModel> failed to load ids: %v", err)
		return
	}

	log.Info("migrate>WorkflowRunOldModel> %d run to migrate", len(ids))
	for _, id := range ids {
		if err := migrateRun(ctx, db, id); err != nil {
			log.Error("unable to migrate run %d: %v", id, err)
		}
	}

	log.Info("End WorkflowRunOldModel migration")
}

func migrateRun(ctx context.Context, db *gorp.DbMap, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	if err := workflow.LockRun(tx, id); err != nil {
		return err
	}

	run, err := workflow.LoadRunByID(tx, id, workflow.LoadRunOptions{})
	if err != nil {
		return err
	}

	if run.Workflow.WorkflowData == nil {
		data := run.Workflow.Migrate(true)
		run.Workflow.WorkflowData = &data
		if err := workflow.UpdateWorkflowRun(ctx, tx, run); err != nil {
			return err
		}
	}

	return sdk.WithStack(tx.Commit())
}
