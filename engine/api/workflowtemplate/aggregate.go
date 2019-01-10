package workflowtemplate

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// AggregateAuditsOnWorkflowTemplate set audits for each workflow template.
func AggregateAuditsOnWorkflowTemplate(db gorp.SqlExecutor, wts ...*sdk.WorkflowTemplate) error {
	as, err := GetAuditsByTemplateIDsAndEventTypesAndVersionGTE(db, sdk.WorkflowTemplatesToIDs(wts),
		[]string{"WorkflowTemplateAdd", "WorkflowTemplateUpdate"}, 0)
	if err != nil {
		return err
	}

	m := map[int64][]sdk.AuditWorkflowTemplate{}
	for _, a := range as {
		if _, ok := m[a.WorkflowTemplateID]; !ok {
			m[a.WorkflowTemplateID] = []sdk.AuditWorkflowTemplate{}
		}
		m[a.WorkflowTemplateID] = append(m[a.WorkflowTemplateID], a)
	}

	// assume that audits are sorted by creation date desc by GetAudits
	for _, wt := range wts {
		if as, ok := m[wt.ID]; ok {
			wt.FirstAudit = &as[len(as)-1]
			wt.LastAudit = &as[0]
		}
	}

	return nil
}

// AggregateAuditsOnWorkflowTemplateInstance set audits for each workflow template instance.
func AggregateAuditsOnWorkflowTemplateInstance(db gorp.SqlExecutor, wtis ...*sdk.WorkflowTemplateInstance) error {
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

// AggregateTemplateInstanceOnWorkflow set template instance data for each workflow.
func AggregateTemplateInstanceOnWorkflow(db gorp.SqlExecutor, ws ...*sdk.Workflow) error {
	if len(ws) == 0 {
		return nil
	}

	wtis, err := GetInstancesByWorkflowIDs(db, sdk.WorkflowToIDs(ws))
	if err != nil {
		return err
	}
	if len(wtis) == 0 {
		return nil
	}

	mWorkflowTemplateInstances := make(map[int64]sdk.WorkflowTemplateInstance, len(wtis))
	for _, wti := range wtis {
		if wti.WorkflowID != nil {
			mWorkflowTemplateInstances[*wti.WorkflowID] = wti
		}
	}

	for _, w := range ws {
		if wti, ok := mWorkflowTemplateInstances[w.ID]; ok {
			w.TemplateInstance = &wti
		}
	}

	return nil
}

// AggregateTemplateOnInstance set template data for each instance.
func AggregateTemplateOnInstance(db gorp.SqlExecutor, wtis ...*sdk.WorkflowTemplateInstance) error {
	if len(wtis) == 0 {
		return nil
	}

	wts, err := GetAllByIDs(db, sdk.WorkflowTemplateInstancesToWorkflowTemplateIDs(wtis))
	if err != nil {
		return err
	}
	if len(wts) == 0 {
		return nil
	}

	mWorkflowTemplates := make(map[int64]sdk.WorkflowTemplate, len(wts))
	for _, wt := range wts {
		mWorkflowTemplates[wt.ID] = wt
	}

	for _, wti := range wtis {
		if wt, ok := mWorkflowTemplates[wti.WorkflowTemplateID]; ok {
			wti.Template = &wt
		}
	}

	return nil
}
