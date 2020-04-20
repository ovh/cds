package migrate

import (
	"context"
	"database/sql"

	"github.com/ovh/cds/engine/api/project"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func CleanDuplicateHooks(ctx context.Context, db *gorp.DbMap, store cache.Store, dryrun bool) error {
	var ids []int64

	if _, err := db.Select(&ids, "select id from workflow"); err != nil {
		return sdk.WrapError(err, "unable to select workflow")
	}

	var mError = new(sdk.MultiError)
	for _, id := range ids {
		if err := cleanDuplicateHooks(ctx, db, store, id, dryrun); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.CleanDuplicateHooks> unable to clean workflow %d: %v", id, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func cleanDuplicateHooks(ctx context.Context, db *gorp.DbMap, store cache.Store, workflowID int64, dryrun bool) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	projectID, err := tx.SelectInt("SELECT project_id FROM workflow WHERE id = $1", workflowID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WithStack(err)
	}

	if projectID == 0 {
		return nil
	}

	proj, err := project.LoadByID(tx, projectID,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithIntegrations)
	if err != nil {
		return sdk.WrapError(err, "unable to load project %d", projectID)
	}

	w, err := workflow.LoadAndLockByID(ctx, tx, store, *proj, workflowID, workflow.LoadOptions{})
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil
		}
		return err
	}

	if w.FromRepository != "" {
		return nil
	}

	if w.FromTemplate != "" {
		return nil
	}

	nbHooks := len(w.WorkflowData.Node.Hooks)
	if nbHooks < 2 {
		return nil
	}

	var hookDoublons = []struct {
		x int
		y int
	}{}

	for i, h1 := range w.WorkflowData.Node.Hooks {
		for j, h2 := range w.WorkflowData.Node.Hooks {
			if i != j && i < j && h1.Ref() == h2.Ref() {
				hookDoublons = append(hookDoublons, struct{ x, y int }{i, j})
			}
		}
	}

	if len(hookDoublons) == 0 {
		return nil
	}

	var idxToRemove []int64
	for _, doublon := range hookDoublons {
		h1 := w.WorkflowData.Node.Hooks[doublon.x]
		h2 := w.WorkflowData.Node.Hooks[doublon.y]
		if h1.ID < h2.ID {
			idxToRemove = append(idxToRemove, int64(doublon.y))
		} else {
			idxToRemove = append(idxToRemove, int64(doublon.x))
		}
	}

	var newHooks []sdk.NodeHook
	for i, h := range w.WorkflowData.Node.Hooks {
		if !sdk.IsInInt64Array(int64(i), idxToRemove) {
			newHooks = append(newHooks, h)
		}
	}
	w.WorkflowData.Node.Hooks = newHooks

	if err := workflow.Update(ctx, tx, store, *proj, w, workflow.UpdateOptions{DisableHookManagement: dryrun}); err != nil {
		return err
	}

	if dryrun {
		log.Info(ctx, "migrate.cleanDuplicateHooks> workflow %s/%s (%d) should been cleaned", proj.Name, w.Name, w.ID)
	} else {
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Info(ctx, "migrate.cleanDuplicateHooks> workflow %s/%s (%d) has been cleaned", proj.Name, w.Name, w.ID)
	}

	return nil
}

func FixEmptyUUIDHooks(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	q := "select distinct(workflow.id) from w_node_hook join w_node on w_node.id = w_node_hook.node_id  join workflow on workflow.id = w_node.workflow_id  where uuid = ''"
	var ids []int64

	if _, err := db.Select(&ids, q); err != nil {
		return sdk.WrapError(err, "unable to select workflow")
	}

	var mError = new(sdk.MultiError)
	for _, id := range ids {
		if err := fixEmptyUUIDHooks(ctx, db, store, id); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.FixEmptyUUIDHooks> unable to clean workflow %d: %v", id, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func fixEmptyUUIDHooks(ctx context.Context, db *gorp.DbMap, store cache.Store, workflowID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	projectID, err := tx.SelectInt("SELECT project_id FROM workflow WHERE id = $1", workflowID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WithStack(err)
	}

	if projectID == 0 {
		return nil
	}

	proj, err := project.LoadByID(tx, projectID,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithIntegrations)
	if err != nil {
		return sdk.WrapError(err, "unable to load project %d", projectID)
	}

	w, err := workflow.LoadAndLockByID(ctx, tx, store, *proj, workflowID, workflow.LoadOptions{})
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil
		}
		return err
	}

	for i, h := range w.WorkflowData.Node.Hooks {
		if h.UUID == "" {
			w.WorkflowData.Node.Hooks[i].UUID = sdk.UUID()
		}
	}

	if err := workflow.Update(ctx, tx, store, *proj, w, workflow.UpdateOptions{}); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	log.Info(ctx, "migrate.fixEmptyUUIDHooks> workflow %s/%s (%d) has been cleaned", proj.Name, w.Name, w.ID)

	return nil
}
