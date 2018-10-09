package workflowtemplate

import (
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

// GetAll returns all workflow templates for given criteria.
func GetAll(db *gorp.DbMap, c Criteria) ([]sdk.WorkflowTemplate, error) {
	wts := []sdk.WorkflowTemplate{}

	if _, err := db.Select(&wts, fmt.Sprintf("SELECT * FROM workflow_template WHERE %s", c.where()), c.args()); err != nil {
		if err == sql.ErrNoRows {
			err = sdk.NewError(sdk.ErrNotFound, err)
		}
		return nil, sdk.WrapError(err, "Cannot get workflows")
	}

	return wts, nil
}

// Get returns the workflow template for given criteria.
func Get(db gorp.SqlExecutor, c Criteria) (*sdk.WorkflowTemplate, error) {
	w := sdk.WorkflowTemplate{}

	if err := db.SelectOne(&w, fmt.Sprintf("SELECT * FROM workflow_template WHERE %s", c.where()), c.args()); err != nil {
		if err == sql.ErrNoRows {
			err = sdk.NewError(sdk.ErrNotFound, err)
		}
		return nil, sdk.WrapError(err, "Cannot get workflow")
	}

	return &w, nil
}

// Insert template in database.
func Insert(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate) error {
	return sdk.WrapError(database.Insert(db, wt), "Unable to insert workflow template %s", wt.Name)
}

// Update template in database.
func Update(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate) error {
	return sdk.WrapError(database.Update(db, wt), "Unable to update workflow template %s", wt.Name)
}

// InsertRelation between workflow template and workflow in database.
func InsertRelation(db gorp.SqlExecutor, wtw *sdk.WorkflowTemplateWorkflow) error {
	return sdk.WrapError(database.Insert(db, wtw), "Unable to insert workflow template relation %d with workflow %d",
		wtw.WorkflowTemplateID, wtw.WorkflowID)
}

// DeleteRelationsForWorkflowID removes all relation for workflow by id in database.
func DeleteRelationsForWorkflowID(db gorp.SqlExecutor, workflowID int64) error {
	_, err := db.Exec("DELETE FROM workflow_template_workflow WHERE workflow_id = $1", workflowID)
	return sdk.WrapError(err, "Unable to remove all relations for workflow %d", workflowID)
}
