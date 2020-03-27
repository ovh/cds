package workflowtemplate

import (
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// CreateAuditAdd create an audit for template add.
func CreateAuditAdd(db gorp.SqlExecutor, wt sdk.WorkflowTemplate, u sdk.Identifiable) error {
	return InsertAudit(db, &sdk.AuditWorkflowTemplate{
		AuditCommon: sdk.AuditCommon{
			EventType:   "WorkflowTemplateAdd",
			Created:     time.Now(),
			TriggeredBy: u.GetUsername(),
		},
		WorkflowTemplateID: wt.ID,
		DataAfter:          wt,
	})
}

// CreateAuditUpdate create an audit for template update.
func CreateAuditUpdate(db gorp.SqlExecutor, oldT, newT sdk.WorkflowTemplate, changeMessage string, u sdk.Identifiable) error {
	return InsertAudit(db, &sdk.AuditWorkflowTemplate{
		AuditCommon: sdk.AuditCommon{
			EventType:   "WorkflowTemplateUpdate",
			Created:     time.Now(),
			TriggeredBy: u.GetUsername(),
		},
		WorkflowTemplateID: newT.ID,
		ChangeMessage:      changeMessage,
		DataBefore:         oldT,
		DataAfter:          newT,
	})
}

// CreateAuditInstanceAdd create an audit for template instance add.
func CreateAuditInstanceAdd(db gorp.SqlExecutor, wti sdk.WorkflowTemplateInstance, u sdk.Identifiable) error {
	b, err := json.Marshal(wti)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal workflow template instance")
	}
	return InsertInstanceAudit(db, &sdk.AuditWorkflowTemplateInstance{
		AuditCommon: sdk.AuditCommon{
			EventType:   "WorkflowTemplateInstanceAdd",
			Created:     time.Now(),
			TriggeredBy: u.GetUsername(),
		},
		WorkflowTemplateInstanceID: wti.ID,
		DataType:                   "json",
		DataAfter:                  string(b),
	})
}

// CreateAuditInstanceUpdate create an audit for template instance update.
func CreateAuditInstanceUpdate(db gorp.SqlExecutor, oldI, newI sdk.WorkflowTemplateInstance, u sdk.Identifiable) error {
	before, err := json.Marshal(oldI)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal workflow template instance")
	}
	after, err := json.Marshal(newI)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal workflow template instance")
	}
	return InsertInstanceAudit(db, &sdk.AuditWorkflowTemplateInstance{
		AuditCommon: sdk.AuditCommon{
			EventType:   "WorkflowTemplateInstanceUpdate",
			Created:     time.Now(),
			TriggeredBy: u.GetUsername(),
		},
		WorkflowTemplateInstanceID: newI.ID,
		DataType:                   "json",
		DataBefore:                 string(before),
		DataAfter:                  string(after),
	})
}
