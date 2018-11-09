package workflowtemplate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/mitchellh/mapstructure"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	audits = map[string]sdk.Audit{
		fmt.Sprintf("%T", sdk.EventWorkflowTemplateAdd{}):            addWorkflowTemplateAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowTemplateUpdate{}):         updateWorkflowTemplateAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowTemplateInstanceAdd{}):    addWorkflowTemplateInstanceAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowTemplateInstanceUpdate{}): updateWorkflowTemplateInstanceAudit{},
	}
)

// ComputeAudit compute audit on workflow template.
func ComputeAudit(c context.Context, DBFunc func() *gorp.DbMap) {
	chanEvent := make(chan sdk.Event)
	event.Subscribe(chanEvent)

	db := DBFunc()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("%v", sdk.WithStack(c.Err()))
				return
			}
		case e := <-chanEvent:
			if !strings.HasPrefix(e.EventType, "sdk.EventWorkflowTemplate") {
				continue
			}

			if audit, ok := audits[e.EventType]; ok {
				if err := audit.Compute(db, e); err != nil {
					log.Warning("%v", sdk.WrapError(err, "Unable to compute audit on event %s", e.EventType))
				}
			}
		}
	}
}

type addWorkflowTemplateAudit struct{}

func (a addWorkflowTemplateAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var wtEvent sdk.EventWorkflowTemplateAdd
	if err := mapstructure.Decode(e.Payload, &wtEvent); err != nil {
		return sdk.WrapError(err, "Unable to decode payload")
	}

	b, err := json.Marshal(wtEvent.WorkflowTemplate)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal workflow template")
	}

	return InsertAudit(db, &sdk.AuditWorkflowTemplate{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
			DataAfter:   string(b),
			DataType:    "json",
		},
		WorkflowTemplateID: wtEvent.WorkflowTemplate.ID,
	})
}

type updateWorkflowTemplateAudit struct{}

func (a updateWorkflowTemplateAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var wtEvent sdk.EventWorkflowTemplateUpdate
	if err := mapstructure.Decode(e.Payload, &wtEvent); err != nil {
		return sdk.WrapError(err, "Unable to decode payload")
	}

	before, err := json.Marshal(wtEvent.NewWorkflowTemplate)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal workflow template")
	}

	after, err := json.Marshal(wtEvent.OldWorkflowTemplate)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal workflow template")
	}

	return InsertAudit(db, &sdk.AuditWorkflowTemplate{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
			DataBefore:  string(before),
			DataAfter:   string(after),
			DataType:    "json",
		},
		WorkflowTemplateID: wtEvent.NewWorkflowTemplate.ID,
	})
}

type addWorkflowTemplateInstanceAudit struct{}

func (a addWorkflowTemplateInstanceAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var wtEvent sdk.EventWorkflowTemplateInstanceAdd
	if err := mapstructure.Decode(e.Payload, &wtEvent); err != nil {
		return sdk.WrapError(err, "Unable to decode payload")
	}

	b, err := json.Marshal(wtEvent.WorkflowTemplateInstance)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal workflow template instance")
	}

	return InsertInstanceAudit(db, &sdk.AuditWorkflowTemplateInstance{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
			DataAfter:   string(b),
			DataType:    "json",
		},
		WorkflowTemplateInstanceID: wtEvent.WorkflowTemplateInstance.ID,
	})
}

type updateWorkflowTemplateInstanceAudit struct{}

func (a updateWorkflowTemplateInstanceAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var wtEvent sdk.EventWorkflowTemplateInstanceUpdate
	if err := mapstructure.Decode(e.Payload, &wtEvent); err != nil {
		return sdk.WrapError(err, "Unable to decode payload")
	}

	before, err := json.Marshal(wtEvent.NewWorkflowTemplateInstance)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal workflow template instance")
	}

	after, err := json.Marshal(wtEvent.OldWorkflowTemplateInstance)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal workflow template instance")
	}

	return InsertInstanceAudit(db, &sdk.AuditWorkflowTemplateInstance{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
			DataBefore:  string(before),
			DataAfter:   string(after),
			DataType:    "json",
		},
		WorkflowTemplateInstanceID: wtEvent.NewWorkflowTemplateInstance.ID,
	})
}
