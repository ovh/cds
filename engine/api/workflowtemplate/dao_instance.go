package workflowtemplate

import (
	"context"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getInstance(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadInstanceOptionFunc) (*sdk.WorkflowTemplateInstance, error) {
	var wti sdk.WorkflowTemplateInstance

	found, err := gorpmapping.Get(ctx, db, q, &wti)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get workflow template instance")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	for i := range opts {
		if err := opts[i](ctx, db, &wti); err != nil {
			return nil, err
		}
	}

	return &wti, nil
}

// InsertInstance for workflow template in database.
func InsertInstance(db gorp.SqlExecutor, wti *sdk.WorkflowTemplateInstance) error {
	return sdk.WrapError(gorpmapping.Insert(db, wti), "unable to insert workflow template relation %d with workflow %d",
		wti.WorkflowTemplateID, wti.WorkflowID)
}

// UpdateInstance for workflow template in database.
func UpdateInstance(db gorp.SqlExecutor, wti *sdk.WorkflowTemplateInstance) error {
	return sdk.WrapError(gorpmapping.Update(db, wti), "unable to update workflow template instance %d", wti.ID)
}

// DeleteInstance for workflow template in database.
func DeleteInstance(db gorp.SqlExecutor, wti *sdk.WorkflowTemplateInstance) error {
	return sdk.WrapError(gorpmapping.Delete(db, wti), "unable to delete workflow template instance %d", wti.ID)
}

// DeleteInstanceNotIDAndWorkflowID removes all instances of a template where not id and workflow id equal in database.
func DeleteInstanceNotIDAndWorkflowID(db gorp.SqlExecutor, id, workflowID int64) error {
	_, err := db.Exec("DELETE FROM workflow_template_instance WHERE id != $1 AND workflow_id = $2", id, workflowID)
	return sdk.WrapError(err, "unable to remove all instances for workflow %d", workflowID)
}

// LoadInstancesByTemplateIDAndProjectIDs returns all workflow template instances by template id and project ids.
func LoadInstancesByTemplateIDAndProjectIDs(ctx context.Context, db gorp.SqlExecutor, templateID int64, projectIDs []int64) ([]sdk.WorkflowTemplateInstance, error) {
	wtis := []sdk.WorkflowTemplateInstance{}

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM workflow_template_instance
    WHERE workflow_template_id = $1 AND project_id = ANY(string_to_array($2, ',')::int[])
  `).Args(templateID, gorpmapping.IDsToQueryString(projectIDs))
	if err := gorpmapping.GetAll(ctx, db, query, &wtis); err != nil {
		return nil, sdk.WrapError(err, "cannot get workflow template instances")
	}

	return wtis, nil
}

// LoadInstancesByTemplateIDAndProjectIDAndRequestWorkflowName returns all workflow template instances by template id, project id and request workflow name.
func LoadInstancesByTemplateIDAndProjectIDAndRequestWorkflowName(ctx context.Context, db gorp.SqlExecutor, templateID, projectID int64, workflowName string) ([]sdk.WorkflowTemplateInstance, error) {
	wtis := []sdk.WorkflowTemplateInstance{}

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM workflow_template_instance
    WHERE workflow_template_id = $1 AND project_id = $2 AND (request->>'workflow_name')::text = $3
  `).Args(templateID, projectID, workflowName)
	if err := gorpmapping.GetAll(ctx, db, query, &wtis); err != nil {
		return nil, sdk.WrapError(err, "cannot get workflow template instances")
	}

	return wtis, nil
}

// LoadInstancesByWorkflowIDs returns all workflow template instances by workflow ids.
func LoadInstancesByWorkflowIDs(ctx context.Context, db gorp.SqlExecutor, workflowIDs []int64) ([]sdk.WorkflowTemplateInstance, error) {
	wtis := []sdk.WorkflowTemplateInstance{}

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM workflow_template_instance
    WHERE workflow_id = ANY(string_to_array($1, ',')::int[])
  `).Args(gorpmapping.IDsToQueryString(workflowIDs))
	if err := gorpmapping.GetAll(ctx, db, query, &wtis); err != nil {
		return nil, sdk.WrapError(err, "cannot get workflow template instances")
	}

	return wtis, nil
}

// LoadInstanceByWorkflowID returns a workflow template instance by workflow id.
func LoadInstanceByWorkflowID(ctx context.Context, db gorp.SqlExecutor, workflowID int64, opts ...LoadInstanceOptionFunc) (*sdk.WorkflowTemplateInstance, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM workflow_template_instance
    WHERE workflow_id = $1
  `).Args(workflowID)
	return getInstance(ctx, db, query, opts...)
}

// LoadInstanceByWorkflowNameAndTemplateIDAndProjectID returns a workflow template instance by workflow name, template id and project id.
func LoadInstanceByWorkflowNameAndTemplateIDAndProjectID(ctx context.Context, db gorp.SqlExecutor, workflowName string, templateID, projectID int64, opts ...LoadInstanceOptionFunc) (*sdk.WorkflowTemplateInstance, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM workflow_template_instance
    WHERE workflow_name = $1 AND workflow_template_id = $2 AND project_id = $3
  `).Args(workflowName, templateID, projectID)
	return getInstance(ctx, db, query, opts...)
}

// LoadInstanceByIDForTemplateIDAndProjectIDs returns a workflow template instance by id, template id in project ids.
func LoadInstanceByIDForTemplateIDAndProjectIDs(ctx context.Context, db gorp.SqlExecutor, id, templateID int64, projectIDs []int64, opts ...LoadInstanceOptionFunc) (*sdk.WorkflowTemplateInstance, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM workflow_template_instance
    WHERE id = $1 AND workflow_template_id = $2 AND project_id = ANY(string_to_array($3, ',')::int[])
  `).Args(id, templateID, gorpmapping.IDsToQueryString(projectIDs))
	return getInstance(ctx, db, query, opts...)
}

// InsertInstanceAudit for workflow template instance in database.
func InsertInstanceAudit(db gorp.SqlExecutor, awti *sdk.AuditWorkflowTemplateInstance) error {
	return sdk.WrapError(gorpmapping.Insert(db, awti), "unable to insert audit for workflow template instance %d", awti.WorkflowTemplateInstanceID)
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
		return nil, sdk.WrapError(err, "cannot get workflow template instance audits")
	}

	return awtis, nil
}
