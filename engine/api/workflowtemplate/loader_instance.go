package workflowtemplate

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadInstanceOptionFunc for workflow template instance.
type LoadInstanceOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.WorkflowTemplateInstance) error

// LoadInstanceOptions provides all options on workflow template instance loads functions
var LoadInstanceOptions = struct {
	WithTemplate LoadInstanceOptionFunc
	WithAudits   LoadInstanceOptionFunc
}{
	WithTemplate: loadInstanceTemplate,
	WithAudits:   loadInstanceAudits,
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

func loadInstanceAudits(ctx context.Context, db gorp.SqlExecutor, wtis ...*sdk.WorkflowTemplateInstance) error {
	as, err := GetInstanceAuditsByInstanceIDsAndEventTypes(db,
		sdk.WorkflowTemplateInstancesToIDs(wtis),
		[]string{"WorkflowTemplateInstanceAdd", "WorkflowTemplateInstanceUpdate"},
	)
	if err != nil {
		return err
	}

	m := map[int64][]sdk.AuditWorkflowTemplateInstance{}
	for _, a := range as {
		if _, ok := m[a.WorkflowTemplateInstanceID]; !ok {
			m[a.WorkflowTemplateInstanceID] = []sdk.AuditWorkflowTemplateInstance{}
		}
		m[a.WorkflowTemplateInstanceID] = append(m[a.WorkflowTemplateInstanceID], a)
	}

	// assume that audits are sorted by creation date with GetInstanceAudits
	for _, wti := range wtis {
		if as, ok := m[wti.ID]; ok {
			wti.FirstAudit = &as[0]
			wti.LastAudit = &as[len(as)-1]
		}
	}

	return nil
}
