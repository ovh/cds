package workflowtemplate

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc for workflow template.
type LoadOptionFunc func(gorp.SqlExecutor, ...*sdk.WorkflowTemplate) error

// LoadOptions provides all options to load workflow template.
var LoadOptions = struct {
	Default    LoadOptionFunc
	WithAudits LoadOptionFunc
	WithGroup  LoadOptionFunc
}{
	Default:    loadDefault,
	WithAudits: loadAudits,
	WithGroup:  loadGroup,
}

func loadDefault(db gorp.SqlExecutor, wts ...*sdk.WorkflowTemplate) error {
	return loadGroup(db, wts...)
}

func loadAudits(db gorp.SqlExecutor, wts ...*sdk.WorkflowTemplate) error {
	for i := range wts {
		latestAudit, err := GetAuditLatestByTemplateID(db, wts[i].ID)
		if err != nil {
			return err
		}

		oldestAudit, err := GetAuditOldestByTemplateID(db, wts[i].ID)
		if err != nil {
			return err
		}

		wts[i].FirstAudit = oldestAudit
		wts[i].LastAudit = latestAudit
	}

	return nil
}

func loadGroup(db gorp.SqlExecutor, wts ...*sdk.WorkflowTemplate) error {
	gs := []sdk.Group{}

	if err := gorpmapping.GetAll(db,
		gorpmapping.NewQuery(`SELECT * FROM "group" WHERE id = ANY(string_to_array($1, ',')::int[])`).
			Args(gorpmapping.IDsToQueryString(sdk.WorkflowTemplatesToGroupIDs(wts))),
		&gs,
	); err != nil {
		return sdk.WrapError(err, "cannot get groups")
	}

	m := make(map[int64]sdk.Group, len(gs))
	for i := range gs {
		m[gs[i].ID] = gs[i]
	}

	for _, wt := range wts {
		if g, ok := m[wt.GroupID]; ok {
			wt.Group = &g
		}
	}

	return nil
}

// LoadInstanceOptionFunc for workflow template instance.
type LoadInstanceOptionFunc func(gorp.SqlExecutor, ...*sdk.WorkflowTemplateInstance) error

// LoadInstanceOptions provides all options on workflow template instance loads functions
var LoadInstanceOptions = struct {
	WithTemplate LoadInstanceOptionFunc
}{
	WithTemplate: loadInstanceTemplate,
}

func loadInstanceTemplate(db gorp.SqlExecutor, wtis ...*sdk.WorkflowTemplateInstance) error {
	if len(wtis) == 0 {
		return nil
	}

	wts, err := LoadAllByIDs(db, sdk.WorkflowTemplateInstancesToWorkflowTemplateIDs(wtis), LoadOptions.WithGroup)
	if err != nil {
		return err
	}
	if len(wts) == 0 {
		return nil
	}

	m := make(map[int64]sdk.WorkflowTemplate, len(wts))
	for _, wt := range wts {
		m[wt.ID] = wt
	}

	for _, wti := range wtis {
		if wt, ok := m[wti.WorkflowTemplateID]; ok {
			wti.Template = &wt
		}
	}

	return nil
}
