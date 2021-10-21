package workerhook

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func init() {
	gorpmapping.Register(gorpmapping.New(sdk.WorkerHookProjectIntegrationModel{}, "worker_hook_project_integration", true, "id"))
}

func Insert(ctx context.Context, db gorp.SqlExecutor, h *sdk.WorkerHookProjectIntegrationModel) error {
	_, end := telemetry.Span(ctx, "workerhook.Insert")
	defer end()
	return gorpmapping.Insert(db, h)
}

func Update(ctx context.Context, db gorp.SqlExecutor, h *sdk.WorkerHookProjectIntegrationModel) error {
	_, end := telemetry.Span(ctx, "workerhook.Update")
	defer end()
	return gorpmapping.Update(db, h)
}

func LoadByID(ctx context.Context, db gorp.SqlExecutor, id int64) (*sdk.WorkerHookProjectIntegrationModel, error) {
	ctx, end := telemetry.Span(ctx, "workerhook.LoadByID")
	defer end()

	query := gorpmapping.NewQuery("select * from worker_hook_project_integration where id = $1").Args(id)
	var res = new(sdk.WorkerHookProjectIntegrationModel)
	found, err := gorpmapping.Get(ctx, db, query, res)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return res, nil
}

func LoadByProjectIntegrationID(ctx context.Context, db gorp.SqlExecutor, projectIntegrationID int64) (*sdk.WorkerHookProjectIntegrationModel, error) {
	ctx, end := telemetry.Span(ctx, "workerhook.LoadByProjectIntegrationID")
	defer end()
	query := gorpmapping.NewQuery("select * from worker_hook_project_integration where project_integration_id = $1").Args(projectIntegrationID)
	var res sdk.WorkerHookProjectIntegrationModel
	found, err := gorpmapping.Get(ctx, db, query, &res)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &res, nil
}

func LoadEnabledByProjectIntegrationID(ctx context.Context, db gorp.SqlExecutor, projectIntegrationID int64) (*sdk.WorkerHookProjectIntegrationModel, error) {
	ctx, end := telemetry.Span(ctx, "workerhook.LoadEnabledByProjectIntegrationID")
	defer end()
	query := gorpmapping.NewQuery("select * from worker_hook_project_integration where disable = false and project_integration_id = $1").Args(projectIntegrationID)
	var res sdk.WorkerHookProjectIntegrationModel
	found, err := gorpmapping.Get(ctx, db, query, &res)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &res, nil
}

func LoadAll(ctx context.Context, db gorp.SqlExecutor) ([]sdk.WorkerHookProjectIntegrationModel, error) {
	ctx, end := telemetry.Span(ctx, "workerhook.LoadAll")
	defer end()
	query := gorpmapping.NewQuery("select * from worker_hook_project_integration")
	var res []sdk.WorkerHookProjectIntegrationModel
	err := gorpmapping.GetAll(ctx, db, query, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func DeleteByID(ctx context.Context, db gorp.SqlExecutor, id int64) error {
	_, end := telemetry.Span(ctx, "workerhook.DeleteByID")
	defer end()
	query := "delete from worker_hook_project_integration where id = $1"
	_, err := db.Exec(query, id)
	if err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
