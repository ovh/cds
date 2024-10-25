package workflowtemplate

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// InsertBulk task for workflow template in database.
func InsertBulk(db gorp.SqlExecutor, wtb *sdk.WorkflowTemplateBulk) error {
	return sdk.WrapError(gorpmapping.Insert(db, wtb), "unable to insert workflow template bulk task for template %d",
		wtb.WorkflowTemplateID)
}

// UpdateBulk task for workflow template in database.
func UpdateBulk(db gorp.SqlExecutor, wtb *sdk.WorkflowTemplateBulk) error {
	return sdk.WrapError(gorpmapping.Update(db, wtb), "unable to update workflow template bulk task %d", wtb.ID)
}

// GetBulkByIDAndTemplateID returns the workflow template bulk for given id and template id.
func GetBulkByIDAndTemplateID(ctx context.Context, db gorp.SqlExecutor, id, templateID int64) (*sdk.WorkflowTemplateBulk, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM workflow_template_bulk
    WHERE id = $1 AND workflow_template_id = $2
  `).Args(id, templateID)
	return getBulk(ctx, db, query)
}

// GetAndLockBulkByID returns an bulk from database for given id.
func GetAndLockBulkByID(ctx context.Context, db gorpmapper.SqlExecutorWithTx, id int64) (*sdk.WorkflowTemplateBulk, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_template_bulk WHERE id = $1 FOR UPDATE SKIP LOCKED").Args(id)
	return getBulk(ctx, db, query)
}

func getBulk(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...gorpmapper.GetOptionFunc) (*sdk.WorkflowTemplateBulk, error) {
	var b sdk.WorkflowTemplateBulk
	found, err := gorpmapping.Get(ctx, db, q, &b, opts...)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get template bulk")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &b, nil
}

// GetBulksPending returns the workflow template bulks with pending status.
func GetBulksPending(ctx context.Context, db gorp.SqlExecutor) ([]sdk.WorkflowTemplateBulk, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM workflow_template_bulk
    WHERE status = $1
    LIMIT 5
  `).Args(sdk.OperationStatusPending)
	return getBulks(ctx, db, query)
}

func getBulks(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...gorpmapper.GetOptionFunc) ([]sdk.WorkflowTemplateBulk, error) {
	bs := []sdk.WorkflowTemplateBulk{}
	if err := gorpmapping.GetAll(ctx, db, q, &bs, opts...); err != nil {
		return nil, sdk.WrapError(err, "cannot get template bulks")
	}
	return bs, nil
}
