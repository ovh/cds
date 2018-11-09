package workflowtemplate

import (
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// GetAll returns all workflow templates for given criteria.
func GetAll(db gorp.SqlExecutor, c Criteria) ([]*sdk.WorkflowTemplate, error) {
	wts := []*sdk.WorkflowTemplate{}

	if _, err := db.Select(&wts, fmt.Sprintf("SELECT * FROM workflow_template WHERE %s", c.where()), c.args()); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflow templates")
	}

	return wts, nil
}

// Get returns the workflow template for given criteria.
func Get(db gorp.SqlExecutor, c Criteria) (*sdk.WorkflowTemplate, error) {
	w := sdk.WorkflowTemplate{}

	if err := db.SelectOne(&w, fmt.Sprintf("SELECT * FROM workflow_template WHERE %s", c.where()), c.args()); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Cannot get workflow template")
	}

	return &w, nil
}

// Insert template in database.
func Insert(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate) error {
	return sdk.WrapError(gorpmapping.Insert(db, wt), "Unable to insert workflow template %s", wt.Name)
}

// Update template in database.
func Update(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate) error {
	return sdk.WrapError(gorpmapping.Update(db, wt), "Unable to update workflow template %s", wt.Name)
}

// Delete template in database.
func Delete(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate) error {
	return sdk.WrapError(gorpmapping.Delete(db, wt), "Unable to delete workflow template %s", wt.Name)
}

// InsertAudit for workflow template in database.
func InsertAudit(db gorp.SqlExecutor, awt *sdk.AuditWorkflowTemplate) error {
	return sdk.WrapError(gorpmapping.Insert(db, awt), "Unable to insert audit for workflow template %d", awt.WorkflowTemplateID)
}

// GetAudits returns all workflow template audits for given criteria.
func GetAudits(db gorp.SqlExecutor, c CriteriaAudit) ([]*sdk.AuditWorkflowTemplate, error) {
	awts := []*sdk.AuditWorkflowTemplate{}

	if _, err := db.Select(&awts, fmt.Sprintf("SELECT * FROM workflow_template_audit WHERE %s ORDER BY created ASC", c.where()), c.args()); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflow template audits")
	}

	return awts, nil
}

// InsertInstance for workflow template in database.
func InsertInstance(db gorp.SqlExecutor, wti *sdk.WorkflowTemplateInstance) error {
	return sdk.WrapError(gorpmapping.Insert(db, wti), "Unable to insert workflow template relation %d with workflow %d",
		wti.WorkflowTemplateID, wti.WorkflowID)
}

// UpdateInstance for workflow template in database.
func UpdateInstance(db gorp.SqlExecutor, wti *sdk.WorkflowTemplateInstance) error {
	return sdk.WrapError(gorpmapping.Update(db, wti), "Unable to update workflow template instance %d", wti.ID)
}

// DeleteInstanceNotIDAndWorkflowID removes all instances of a template where not id and workflow id equal in database.
func DeleteInstanceNotIDAndWorkflowID(db gorp.SqlExecutor, id, workflowID int64) error {
	_, err := db.Exec("DELETE FROM workflow_template_instance WHERE id != $1 AND workflow_id = $2", id, workflowID)
	return sdk.WrapError(err, "Unable to remove all instances for workflow %d", workflowID)
}

// DeleteInstancesForWorkflowTemplateID removes all template instances by template id in database.
func DeleteInstancesForWorkflowTemplateID(db gorp.SqlExecutor, workflowTemplateID int64) error {
	_, err := db.Exec("DELETE FROM workflow_template_instance WHERE workflow_template_id = $1", workflowTemplateID)
	return sdk.WrapError(err, "Unable to remove all instances for workflow template %d", workflowTemplateID)
}

// GetInstances returns all workflow template instances for given criteria.
func GetInstances(db gorp.SqlExecutor, c CriteriaInstance) ([]*sdk.WorkflowTemplateInstance, error) {
	wtis := []*sdk.WorkflowTemplateInstance{}

	if _, err := db.Select(&wtis, fmt.Sprintf("SELECT * FROM workflow_template_instance WHERE %s", c.where()), c.args()); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflow template instances")
	}

	return wtis, nil
}

// GetInstance returns a workflow template instance for given criteria.
func GetInstance(db gorp.SqlExecutor, c CriteriaInstance) (*sdk.WorkflowTemplateInstance, error) {
	wti := sdk.WorkflowTemplateInstance{}

	if err := db.SelectOne(&wti, fmt.Sprintf("SELECT * FROM workflow_template_instance WHERE %s", c.where()), c.args()); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Cannot get workflow template instance")
	}

	return &wti, nil
}

// InsertInstanceAudit for workflow template instance in database.
func InsertInstanceAudit(db gorp.SqlExecutor, awti *sdk.AuditWorkflowTemplateInstance) error {
	return sdk.WrapError(gorpmapping.Insert(db, awti), "Unable to insert audit for workflow template instance %d", awti.WorkflowTemplateInstanceID)
}

// GetInstanceAudits returns all workflow template instance audits for given criteria.
func GetInstanceAudits(db gorp.SqlExecutor, c CriteriaInstanceAudit) ([]*sdk.AuditWorkflowTemplateInstance, error) {
	awtis := []*sdk.AuditWorkflowTemplateInstance{}

	if _, err := db.Select(&awtis, fmt.Sprintf("SELECT * FROM workflow_template_instance_audit WHERE %s ORDER BY created ASC", c.where()), c.args()); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflow template instance audits")
	}

	return awtis, nil
}
