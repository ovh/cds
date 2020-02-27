package workflowtemplate

import (
	"context"

	"github.com/ovh/cds/engine/api/group"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc for workflow template.
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.WorkflowTemplate) error

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

func loadDefault(ctx context.Context, db gorp.SqlExecutor, wts ...*sdk.WorkflowTemplate) error {
	return loadGroup(ctx, db, wts...)
}

func loadAudits(ctx context.Context, db gorp.SqlExecutor, wts ...*sdk.WorkflowTemplate) error {
	for i := range wts {
		latestAudit, err := LoadAuditLatestByTemplateID(ctx, db, wts[i].ID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}

		oldestAudit, err := LoadAuditOldestByTemplateID(ctx, db, wts[i].ID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}

		wts[i].FirstAudit = oldestAudit
		wts[i].LastAudit = latestAudit
	}

	return nil
}

func loadGroup(ctx context.Context, db gorp.SqlExecutor, wts ...*sdk.WorkflowTemplate) error {
	gs, err := group.LoadAllByIDs(ctx, db, sdk.WorkflowTemplatesToGroupIDs(wts))
	if err != nil {
		return err
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
type LoadInstanceOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.WorkflowTemplateInstance) error

// LoadInstanceOptions provides all options on workflow template instance loads functions
var LoadInstanceOptions = struct {
	WithTemplate LoadInstanceOptionFunc
}{
	WithTemplate: loadInstanceTemplate,
}

func loadInstanceTemplate(ctx context.Context, db gorp.SqlExecutor, wtis ...*sdk.WorkflowTemplateInstance) error {
	if len(wtis) == 0 {
		return nil
	}

	wts, err := LoadAllByIDs(ctx, db, sdk.WorkflowTemplateInstancesToWorkflowTemplateIDs(wtis), LoadOptions.WithGroup)
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
