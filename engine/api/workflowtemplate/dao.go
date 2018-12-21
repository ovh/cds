package workflowtemplate

import (
	"database/sql"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// GetAll returns all workflow templates.
func GetAll(db gorp.SqlExecutor) ([]sdk.WorkflowTemplate, error) {
	wts := []sdk.WorkflowTemplate{}

	if _, err := db.Select(&wts, "SELECT * FROM workflow_template"); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflow templates")
	}

	return wts, nil
}

// GetAllByGroupIDs returns all workflow templates by group ids.
func GetAllByGroupIDs(db gorp.SqlExecutor, groupIDs []int64) ([]sdk.WorkflowTemplate, error) {
	wts := []sdk.WorkflowTemplate{}

	if _, err := db.Select(&wts,
		"SELECT * FROM workflow_template WHERE group_id = ANY(string_to_array($1, ',')::int[])",
		gorpmapping.IDsToQueryString(groupIDs),
	); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflow templates")
	}

	return wts, nil
}

// GetAllByIDs returns all workflow templates by ids.
func GetAllByIDs(db gorp.SqlExecutor, ids []int64) ([]sdk.WorkflowTemplate, error) {
	wts := []sdk.WorkflowTemplate{}

	if _, err := db.Select(&wts,
		"SELECT * FROM workflow_template WHERE id = ANY(string_to_array($1, ',')::int[])",
		gorpmapping.IDsToQueryString(ids),
	); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflow templates")
	}

	return wts, nil
}

// GetByID returns the workflow template for given id.
func GetByID(db gorp.SqlExecutor, id int64) (*sdk.WorkflowTemplate, error) {
	w := sdk.WorkflowTemplate{}

	if err := db.SelectOne(&w, "SELECT * FROM workflow_template WHERE id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Cannot get workflow template")
	}

	return &w, nil
}

// GetByIDAndGroupIDs returns the workflow template for given id and group ids.
func GetByIDAndGroupIDs(db gorp.SqlExecutor, id int64, groupIDs []int64) (*sdk.WorkflowTemplate, error) {
	w := sdk.WorkflowTemplate{}

	if err := db.SelectOne(&w,
		"SELECT * FROM workflow_template WHERE id = $1 AND group_id = ANY(string_to_array($2, ',')::int[])",
		id, gorpmapping.IDsToQueryString(groupIDs),
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Cannot get workflow template")
	}

	return &w, nil
}

// GetBySlugAndGroupIDs returns the workflow template for given slug and group ids.
func GetBySlugAndGroupIDs(db gorp.SqlExecutor, slug string, groupIDs []int64) (*sdk.WorkflowTemplate, error) {
	w := sdk.WorkflowTemplate{}

	if err := db.SelectOne(&w,
		"SELECT * FROM workflow_template WHERE slug = $1 AND group_id = ANY(string_to_array($2, ',')::int[])",
		slug, gorpmapping.IDsToQueryString(groupIDs),
	); err != nil {
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

// GetAuditsByTemplateIDsAndEventTypesAndVersionGTE returns all workflow template audits by template ids, event types and version greater or equal.
func GetAuditsByTemplateIDsAndEventTypesAndVersionGTE(db gorp.SqlExecutor, templateIDs []int64, eventTypes []string, version int64) ([]sdk.AuditWorkflowTemplate, error) {
	awts := []sdk.AuditWorkflowTemplate{}

	if _, err := db.Select(&awts,
		`SELECT * FROM workflow_template_audit
     WHERE workflow_template_id = ANY(string_to_array($1, ',')::int[])
     AND event_type = ANY(string_to_array($2, ',')::text[])
     AND (data_after->>'version')::int >= $3
     ORDER BY created DESC`,
		gorpmapping.IDsToQueryString(templateIDs), strings.Join(eventTypes, ","), version,
	); err != nil {
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

// DeleteInstance for workflow template in database.
func DeleteInstance(db gorp.SqlExecutor, wti *sdk.WorkflowTemplateInstance) error {
	return sdk.WrapError(gorpmapping.Delete(db, wti), "Unable to delete workflow template instance %d", wti.ID)
}

// DeleteInstanceNotIDAndWorkflowID removes all instances of a template where not id and workflow id equal in database.
func DeleteInstanceNotIDAndWorkflowID(db gorp.SqlExecutor, id, workflowID int64) error {
	_, err := db.Exec("DELETE FROM workflow_template_instance WHERE id != $1 AND workflow_id = $2", id, workflowID)
	return sdk.WrapError(err, "Unable to remove all instances for workflow %d", workflowID)
}

// GetInstancesByTemplateIDAndProjectIDs returns all workflow template instances by template id and project ids.
func GetInstancesByTemplateIDAndProjectIDs(db gorp.SqlExecutor, templateID int64, projectIDs []int64) ([]sdk.WorkflowTemplateInstance, error) {
	wtis := []sdk.WorkflowTemplateInstance{}

	if _, err := db.Select(&wtis,
		"SELECT * FROM workflow_template_instance WHERE workflow_template_id = $1 AND project_id = ANY(string_to_array($2, ',')::int[])",
		templateID, gorpmapping.IDsToQueryString(projectIDs),
	); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflow template instances")
	}

	return wtis, nil
}

// GetInstancesByTemplateIDAndProjectIDAndWorkflowIDNull returns all workflow template instances by template id, project id and workflow id null.
func GetInstancesByTemplateIDAndProjectIDAndWorkflowIDNull(db gorp.SqlExecutor, templateID, projectID int64) ([]sdk.WorkflowTemplateInstance, error) {
	wtis := []sdk.WorkflowTemplateInstance{}

	if _, err := db.Select(&wtis,
		"SELECT * FROM workflow_template_instance WHERE workflow_id IS NULL AND workflow_template_id = $1 AND project_id = $2",
		templateID, projectID,
	); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflow template instances")
	}

	return wtis, nil
}

// GetInstancesByWorkflowIDs returns all workflow template instances by workflow ids.
func GetInstancesByWorkflowIDs(db gorp.SqlExecutor, workflowIDs []int64) ([]sdk.WorkflowTemplateInstance, error) {
	wtis := []sdk.WorkflowTemplateInstance{}

	if _, err := db.Select(&wtis,
		"SELECT * FROM workflow_template_instance WHERE workflow_id = ANY(string_to_array($1, ',')::int[])",
		gorpmapping.IDsToQueryString(workflowIDs),
	); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflow template instances")
	}

	return wtis, nil
}

// GetInstanceByWorkflowIDAndTemplateID returns a workflow template instance by workflow and template ids.
func GetInstanceByWorkflowIDAndTemplateID(db gorp.SqlExecutor, workflowID, templateID int64) (*sdk.WorkflowTemplateInstance, error) {
	wti := sdk.WorkflowTemplateInstance{}

	if err := db.SelectOne(&wti,
		"SELECT * FROM workflow_template_instance WHERE workflow_id = $1 AND workflow_template_id = $2",
		workflowID, templateID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Cannot get workflow template instance")
	}

	return &wti, nil
}

// GetInstanceByWorkflowID returns a workflow template instance by workflow id.
func GetInstanceByWorkflowID(db gorp.SqlExecutor, workflowID int64) (*sdk.WorkflowTemplateInstance, error) {
	wti := sdk.WorkflowTemplateInstance{}

	if err := db.SelectOne(&wti, "SELECT * FROM workflow_template_instance WHERE workflow_id = $1", workflowID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Cannot get workflow template instance")
	}

	return &wti, nil
}

// GetInstanceByWorkflowNameAndTemplateIDAndProjectID returns a workflow template instance by workflow name, template id and project id.
func GetInstanceByWorkflowNameAndTemplateIDAndProjectID(db gorp.SqlExecutor, workflowName string, templateID, projectID int64) (*sdk.WorkflowTemplateInstance, error) {
	wti := sdk.WorkflowTemplateInstance{}

	if err := db.SelectOne(&wti,
		"SELECT * FROM workflow_template_instance WHERE workflow_name = $1 AND workflow_template_id = $2 AND project_id = $3",
		workflowName, templateID, projectID,
	); err != nil {
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

// GetInstanceAuditsByInstanceIDsAndEventTypes returns all workflow template instance audits by instance ids and event types.
func GetInstanceAuditsByInstanceIDsAndEventTypes(db gorp.SqlExecutor, instanceIDs []int64, eventTypes []string) ([]sdk.AuditWorkflowTemplateInstance, error) {
	awtis := []sdk.AuditWorkflowTemplateInstance{}

	if _, err := db.Select(&awtis,
		`SELECT * FROM workflow_template_instance_audit
     WHERE workflow_template_instance_id = ANY(string_to_array($1, ',')::int[])
     AND event_type = ANY(string_to_array($2, ',')::text[])
     ORDER BY created ASC`,
		gorpmapping.IDsToQueryString(instanceIDs), strings.Join(eventTypes, ","),
	); err != nil {
		return nil, sdk.WrapError(err, "Cannot get workflow template instance audits")
	}

	return awtis, nil
}

// InsertBulk task for workflow template in database.
func InsertBulk(db gorp.SqlExecutor, wtb *sdk.WorkflowTemplateBulk) error {
	return sdk.WrapError(gorpmapping.Insert(db, wtb), "Unable to insert workflow template bulk task for template %d",
		wtb.WorkflowTemplateID)
}

// UpdateBulk task for workflow template in database.
func UpdateBulk(db gorp.SqlExecutor, wtb *sdk.WorkflowTemplateBulk) error {
	return sdk.WrapError(gorpmapping.Update(db, wtb), "Unable to update workflow template bulk task %d", wtb.ID)
}
