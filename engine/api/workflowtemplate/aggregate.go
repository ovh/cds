package workflowtemplate

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

// AggregateAuditsOnWorkflowTemplate set audits for each workflow template.
func AggregateAuditsOnWorkflowTemplate(db *gorp.DbMap, wts ...*sdk.WorkflowTemplate) error {
	as, err := GetAudits(db, NewCriteriaAudit().
		WorkflowTemplateIDs(sdk.WorkflowTemplatesToIDs(wts)...).
		EventTypes("WorkflowTemplateAdd", "WorkflowTemplateUpdate"))
	if err != nil {
		return err
	}

	m := map[int64][]*sdk.AuditWorkflowTemplate{}
	for _, a := range as {
		if _, ok := m[a.WorkflowTemplateID]; !ok {
			m[a.WorkflowTemplateID] = []*sdk.AuditWorkflowTemplate{}
		}
		m[a.WorkflowTemplateID] = append(m[a.WorkflowTemplateID], a)
	}

	// assume that audits are sorted by creation date with GetAudits
	for _, wt := range wts {
		if as, ok := m[wt.ID]; ok {
			wt.FirstAudit = as[0]
			wt.LastAudit = as[len(as)-1]
		}
	}

	return nil
}
