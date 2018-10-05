package workflowtemplate

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

// GetAll returns all existing templates.
func GetAll(db *gorp.DbMap) ([]sdk.WorkflowTemplate, error) {
	wts := []sdk.WorkflowTemplate{}

	if _, err := db.Select(&wts, "SELECT * FROM workflow_templates"); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNotFound // TODO withstack
		}
		return nil, sdk.WrapError(err, "Cannot get workflow")
	}

	return wts, nil
}

// GetByID returns the workflow template for given id.
func GetByID(db gorp.SqlExecutor, id int64) (*sdk.WorkflowTemplate, error) {
	w := sdk.WorkflowTemplate{}

	if err := db.SelectOne(&w, "SELECT * FROM workflow_templates WHERE id=$1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNotFound // TODO withstack
		}
		return nil, sdk.WrapError(err, "Cannot get workflow")
	}

	return &w, nil
}

// InsertWorkflow template in database.
func InsertWorkflow(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate) error {
	return sdk.WrapError(db.Insert(wt), "Unable to insert workflow template %s", wt.Name)
}
