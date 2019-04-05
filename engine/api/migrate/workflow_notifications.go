package migrate

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func WorkflowNotifications(store cache.Store, DBFunc func() *gorp.DbMap) error {
	db := DBFunc()

	log.Info("migrate>WorkflowNotifications> Start migration")

	var ids []int64
	ids, err := workflow.LoadWorkflowIDsWithNotifications(db)
	if err != nil {
		return err
	}

	log.Info("migrate>WorkflowNotifications> %d run to migrate", len(ids))
	for _, id := range ids {
		if err := migrateNotification(db, store, id); err != nil {
			log.Error("cannot migrate notification: %v", err)
			continue
		}
	}

	log.Info("End WorkflowNotifications migration")
	return nil
}

func migrateNotification(db *gorp.DbMap, store cache.Store, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	proj, err := project.LoadProjectByWorkflowID(db, store, nil, id, project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
		project.LoadOptions.WithIntegrations)
	if err != nil {
		return err
	}

	wf, err := workflow.LoadAndLock(tx, id, store, proj, workflow.LoadOptions{}, nil)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrWorkflowNotFound) {
			return nil
		}
		return err
	}

	if err := workflow.DeleteNotifications(tx, wf.ID); err != nil {
		return err
	}
	for _, n := range wf.Notifications {
		if err := workflow.InsertNotification(tx, wf, &n); err != nil {
			return sdk.WrapError(err, "unable to migrate workflow notification %d/%d", wf.ID, n.ID)
		}
	}

	return sdk.WithStack(tx.Commit())
}
