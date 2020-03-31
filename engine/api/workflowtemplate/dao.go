package workflowtemplate

import (
	"context"

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
	a, err := getAudit(ctx, db, query)
	if err != nil {
		return nil, sdk.NewErrorFrom(err, "could not find a template audit with version %d", version)
	}
	return a, nil
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
