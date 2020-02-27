package workflowtemplate

import (
	"context"
	"database/sql"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.WorkflowTemplate, error) {
	pwts := []*sdk.WorkflowTemplate{}

	if err := gorpmapping.GetAll(ctx, db, q, &pwts); err != nil {
		return nil, sdk.WrapError(err, "cannot get workflow templates")
	}
	if len(pwts) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, pwts...); err != nil {
				return nil, err
			}
		}
	}

	wts := make([]sdk.WorkflowTemplate, len(pwts))
	for i := range wts {
		wts[i] = *pwts[i]
	}

	return wts, nil
}

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.WorkflowTemplate, error) {
	var wt sdk.WorkflowTemplate

	found, err := gorpmapping.Get(ctx, db, q, &wt)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get workflow template")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	for i := range opts {
		if err := opts[i](ctx, db, &wt); err != nil {
			return nil, err
		}
	}

	return &wt, nil
}

func getAudit(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (*sdk.AuditWorkflowTemplate, error) {
	var awt sdk.AuditWorkflowTemplate

	found, err := gorpmapping.Get(ctx, db, q, &awt)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get workflow template audit")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &awt, nil
}

// LoadAll workflow templates from database.
func LoadAll(ctx context.Context, db gorp.SqlExecutor, opts ...LoadOptionFunc) ([]sdk.WorkflowTemplate, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_template")
	return getAll(ctx, db, query, opts...)
}

// LoadAllByGroupIDs returns all workflow templates by group ids.
func LoadAllByGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64, opts ...LoadOptionFunc) ([]sdk.WorkflowTemplate, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_template WHERE group_id = ANY(string_to_array($1, ',')::int[])").
		Args(gorpmapping.IDsToQueryString(groupIDs))
	return getAll(ctx, db, query, opts...)
}

// LoadAllByIDs returns all workflow templates by ids.
func LoadAllByIDs(ctx context.Context, db gorp.SqlExecutor, ids []int64, opts ...LoadOptionFunc) ([]sdk.WorkflowTemplate, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_template WHERE id = ANY(string_to_array($1, ',')::int[])").
		Args(gorpmapping.IDsToQueryString(ids))
	return getAll(ctx, db, query, opts...)
}

// LoadByID retrieves in database the workflow template with given id.
func LoadByID(ctx context.Context, db gorp.SqlExecutor, id int64, opts ...LoadOptionFunc) (*sdk.WorkflowTemplate, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_template WHERE id = $1").Args(id)
	return get(ctx, db, query, opts...)
}

// LoadBySlugAndGroupID returns the workflow template for given slug and group id.
func LoadBySlugAndGroupID(ctx context.Context, db gorp.SqlExecutor, slug string, groupID int64, opts ...LoadOptionFunc) (*sdk.WorkflowTemplate, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_template WHERE slug = $1 AND group_id = $2").Args(slug, groupID)
	return get(ctx, db, query, opts...)
}

// Insert template in database.
func Insert(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate) error {
	return sdk.WrapError(gorpmapping.Insert(db, wt), "unable to insert workflow template %s", wt.Name)
}

// Update template in database.
func Update(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate) error {
	return sdk.WrapError(gorpmapping.Update(db, wt), "unable to update workflow template %s", wt.Name)
}

// Delete template in database.
func Delete(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate) error {
	return sdk.WrapError(gorpmapping.Delete(db, wt), "unable to delete workflow template %s", wt.Name)
}

// InsertAudit for workflow template in database.
func InsertAudit(db gorp.SqlExecutor, awt *sdk.AuditWorkflowTemplate) error {
	return sdk.WrapError(gorpmapping.Insert(db, awt), "unable to insert audit for workflow template %d", awt.WorkflowTemplateID)
}

// LoadAuditsByTemplateIDAndVersionGTE returns all workflow template audits by template id and version greater or equal.
func LoadAuditsByTemplateIDAndVersionGTE(db gorp.SqlExecutor, templateID, version int64) ([]sdk.AuditWorkflowTemplate, error) {
	awts := []sdk.AuditWorkflowTemplate{}

	if _, err := db.Select(&awts,
		`SELECT * FROM workflow_template_audit
     WHERE workflow_template_id = $1
     AND (data_after->>'version')::int >= $2
     ORDER BY created DESC`,
		templateID, version,
	); err != nil {
		return nil, sdk.WrapError(err, "cannot get workflow template audits")
	}

	return awts, nil
}

// LoadAuditLatestByTemplateID returns workflow template latest audit by template id.
func LoadAuditLatestByTemplateID(ctx context.Context, db gorp.SqlExecutor, templateID int64) (*sdk.AuditWorkflowTemplate, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM workflow_template_audit
    WHERE workflow_template_id = $1
    ORDER BY created DESC
    LIMIT 1
  `).Args(templateID)
	return getAudit(ctx, db, query)
}

// LoadAuditByTemplateIDAndVersion returns workflow template audit by template id and version.
func LoadAuditByTemplateIDAndVersion(ctx context.Context, db gorp.SqlExecutor, templateID, version int64) (*sdk.AuditWorkflowTemplate, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM workflow_template_audit
    WHERE workflow_template_id = $1 AND (data_after->>'version')::int = $2
  `).Args(templateID, version)
	return getAudit(ctx, db, query)
}

// LoadAuditOldestByTemplateID returns workflow template oldtest audit by template id.
func LoadAuditOldestByTemplateID(ctx context.Context, db gorp.SqlExecutor, templateID int64) (*sdk.AuditWorkflowTemplate, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM workflow_template_audit
    WHERE workflow_template_id = $1
    ORDER BY created ASC
    LIMIT 1
  `).Args(templateID)
	return getAudit(ctx, db, query)
}

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

// GetInstancesByTemplateIDAndProjectIDs returns all workflow template instances by template id and project ids.
func GetInstancesByTemplateIDAndProjectIDs(db gorp.SqlExecutor, templateID int64, projectIDs []int64) ([]sdk.WorkflowTemplateInstance, error) {
	wtis := []sdk.WorkflowTemplateInstance{}

	if _, err := db.Select(&wtis,
		"SELECT * FROM workflow_template_instance WHERE workflow_template_id = $1 AND project_id = ANY(string_to_array($2, ',')::int[])",
		templateID, gorpmapping.IDsToQueryString(projectIDs),
	); err != nil {
		return nil, sdk.WrapError(err, "cannot get workflow template instances")
	}

	return wtis, nil
}

// GetInstancesByTemplateIDAndProjectIDAndRequestWorkflowName returns all workflow template instances by template id, project id and request workflow name.
func GetInstancesByTemplateIDAndProjectIDAndRequestWorkflowName(db gorp.SqlExecutor, templateID, projectID int64, workflowName string) ([]sdk.WorkflowTemplateInstance, error) {
	wtis := []sdk.WorkflowTemplateInstance{}

	if _, err := db.Select(&wtis,
		"SELECT * FROM workflow_template_instance WHERE workflow_template_id = $1 AND project_id = $2 AND (request->>'workflow_name')::text = $3",
		templateID, projectID, workflowName,
	); err != nil {
		return nil, sdk.WrapError(err, "cannot get workflow template instances")
	}

	return wtis, nil
}

// GetInstancesByWorkflowIDs returns all workflow template instances by workflow ids.
func GetInstancesByWorkflowIDs(ctx context.Context, db gorp.SqlExecutor, workflowIDs []int64) ([]sdk.WorkflowTemplateInstance, error) {
	wtis := []sdk.WorkflowTemplateInstance{}

	if _, err := db.Select(&wtis,
		"SELECT * FROM workflow_template_instance WHERE workflow_id = ANY(string_to_array($1, ',')::int[])",
		gorpmapping.IDsToQueryString(workflowIDs),
	); err != nil {
		return nil, sdk.WrapError(err, "cannot get workflow template instances")
	}

	return wtis, nil
}

// LoadInstanceByWorkflowID returns a workflow template instance by workflow id.
func LoadInstanceByWorkflowID(ctx context.Context, db gorp.SqlExecutor, workflowID int64, opts ...LoadInstanceOptionFunc) (*sdk.WorkflowTemplateInstance, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_template_instance WHERE workflow_id = $1").Args(workflowID)
	return getInstance(ctx, db, query, opts...)
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
		return nil, sdk.WrapError(err, "cannot get workflow template instance")
	}

	return &wti, nil
}

// GetInstanceByIDForTemplateIDAndProjectIDs returns a workflow template instance by id, template id in project ids.
func GetInstanceByIDForTemplateIDAndProjectIDs(db gorp.SqlExecutor, id, templateID int64, projectIDs []int64) (*sdk.WorkflowTemplateInstance, error) {
	wti := sdk.WorkflowTemplateInstance{}

	if err := db.SelectOne(&wti,
		"SELECT * FROM workflow_template_instance WHERE id = $1 AND workflow_template_id = $2 AND project_id = ANY(string_to_array($3, ',')::int[])",
		id, templateID, gorpmapping.IDsToQueryString(projectIDs),
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "cannot get workflow template instance")
	}

	return &wti, nil
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
