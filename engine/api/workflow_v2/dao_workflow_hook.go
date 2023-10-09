package workflow_v2

import (
	"context"
	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func getHook(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.V2WorkflowHook, error) {
	var dbHook dbWorkflowHook
	found, err := gorpmapping.Get(ctx, db, query, &dbHook)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WrapError(sdk.ErrNotFound, "unable to find workflow hook")
	}

	isValid, err := gorpmapping.CheckSignature(dbHook, dbHook.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "hook %s: data corrupted", dbHook.ID)
		return nil, sdk.WrapError(sdk.ErrNotFound, "unable to find hook")
	}

	return &dbHook.V2WorkflowHook, nil
}

func getAllHooks(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.V2WorkflowHook, error) {
	var dbHooks []dbWorkflowHook
	if err := gorpmapping.GetAll(ctx, db, query, &dbHooks); err != nil {
		return nil, err
	}
	hooks := make([]sdk.V2WorkflowHook, 0, len(dbHooks))
	for _, h := range dbHooks {
		isValid, err := gorpmapping.CheckSignature(h, h.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "hook %s: data corrupted", h.ID)
			continue
		}
		hooks = append(hooks, h.V2WorkflowHook)
	}
	return hooks, nil
}

func DeleteWorkflowHooks(ctx context.Context, db gorpmapper.SqlExecutorWithTx, entityID string) error {
  _, err := db.Exec("DELETE FROM v2_workflow_hook WHERE entity_id = $1", entityID)
  return sdk.WithStack(err)
}

func InsertWorkflowHook(ctx context.Context, db gorpmapper.SqlExecutorWithTx, h *sdk.V2WorkflowHook) error {
	ctx, next := telemetry.Span(ctx, "workflow_v2.InsertWorkflowHook")
	defer next()
	h.ID = sdk.UUID()
	dbWkfHooks := &dbWorkflowHook{V2WorkflowHook: *h}

	if err := gorpmapping.InsertAndSign(ctx, db, dbWkfHooks); err != nil {
		return err
	}
	*h = dbWkfHooks.V2WorkflowHook
	return nil
}

func LoadHooksByRepositoryEvent(ctx context.Context, db gorp.SqlExecutor, vcsName, repoName, eventName string) ([]sdk.V2WorkflowHook, error) {
	q := gorpmapping.NewQuery(`SELECT * FROM v2_workflow_hook WHERE
    type = $1 AND
    data->>'vcs_server'::text = $2 AND
    data->>'repository_name'::text = $3 AND
    data->>'repository_event'::text = $4`).Args(sdk.WorkflowHookTypeRepository, vcsName, repoName, eventName)
	return getAllHooks(ctx, db, q)
}

func LoadHooksByWorkflowUpdated(ctx context.Context, db gorp.SqlExecutor, projKey, vcsName, repoName, workflowName string) (*sdk.V2WorkflowHook, error) {
	q := gorpmapping.NewQuery(`SELECT * FROM v2_workflow_hook WHERE
    type = $1 AND
    project_key = $2 AND
    vcs_name = $3 AND
    repository_name = $4 AND
    workflow_name = $5`).Args(sdk.WorkflowHookTypeWorkflow, projKey, vcsName, repoName, workflowName)
	return getHook(ctx, db, q)
}

func LoadHooksByModelUpdated(ctx context.Context, db gorp.SqlExecutor, models []string) ([]sdk.V2WorkflowHook, error) {
	q := gorpmapping.NewQuery(`SELECT * FROM v2_workflow_hook WHERE
    type = $1 AND
    data->>'model'::text = ANY($2)`).Args(sdk.WorkflowHookTypeWorkerModel, pq.StringArray(models))
	return getAllHooks(ctx, db, q)
}
