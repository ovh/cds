package workflowtemplate

import (
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

// GetAll returns all workflow templates for given criteria.
func GetAll(db *gorp.DbMap, c Criteria) ([]sdk.WorkflowTemplate, error) {
	wts := []sdk.WorkflowTemplate{}

	if _, err := db.Select(&wts, fmt.Sprintf("SELECT * FROM workflow_templates WHERE %s", c.where()), c.args()); err != nil {
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

	if err := db.SelectOne(&w, fmt.Sprintf("SELECT * FROM workflow_templates WHERE %s", c.where()), c.args()); err != nil {
		if err == sql.ErrNoRows {
			err = sdk.NewError(sdk.ErrNotFound, err)
		}
		return nil, sdk.WrapError(err, "Cannot get workflow")
	}

	return &w, nil
}

// InsertWorkflow template in database.
func InsertWorkflow(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate) error {
	err := db.Insert(wt)
	if e, ok := err.(*pq.Error); ok && e.Code == database.ViolateUniqueKeyPGCode {
		err = sdk.NewError(sdk.ErrConflict, e)
	}
	return sdk.WrapError(err, "Unable to insert workflow template %s", wt.Name)
}
