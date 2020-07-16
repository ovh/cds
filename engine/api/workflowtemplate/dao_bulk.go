package workflowtemplate

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
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
func GetBulkByIDAndTemplateID(db gorp.SqlExecutor, id, templateID int64) (*sdk.WorkflowTemplateBulk, error) {
	b := sdk.WorkflowTemplateBulk{}

	if err := db.SelectOne(&b, `
    SELECT *
    FROM workflow_template_bulk
    WHERE id = $1 AND workflow_template_id = $2
  `, id, templateID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "cannot get workflow template")
	}

	return &b, nil
}
