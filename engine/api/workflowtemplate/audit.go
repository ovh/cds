package workflowtemplate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
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
func ComputeAudit(ctx context.Context, DBFunc func() *gorp.DbMap) {
	chanEvent := make(chan sdk.Event)
	event.Subscribe(chanEvent)

	db := DBFunc()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", sdk.WithStack(ctx.Err()))
				return
			}
		case e := <-chanEvent:
			if !strings.HasPrefix(e.EventType, "sdk.EventWorkflowTemplate") {
				continue
			}

			if audit, ok := audits[e.EventType]; ok {
				if err := audit.Compute(ctx, db, e); err != nil {
					log.Warning(ctx, "%v", sdk.WrapError(err, "Unable to compute audit on event %s", e.EventType))
				}
			}
		}
	}
}

type addWorkflowTemplateAudit struct{}

func (a addWorkflowTemplateAudit) Compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error {
	var wtEvent sdk.EventWorkflowTemplateAdd
	if err := json.Unmarshal(e.Payload, &wtEvent); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal payload")
	}

	return InsertAudit(db, &sdk.AuditWorkflowTemplate{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
		},
		WorkflowTemplateID: wtEvent.WorkflowTemplate.ID,
		DataAfter:          wtEvent.WorkflowTemplate,
	})
}

type updateWorkflowTemplateAudit struct{}

func (a updateWorkflowTemplateAudit) Compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error {
	var wtEvent sdk.EventWorkflowTemplateUpdate
	if err := json.Unmarshal(e.Payload, &wtEvent); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal payload")
	}

	return InsertAudit(db, &sdk.AuditWorkflowTemplate{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
		},
		WorkflowTemplateID: wtEvent.NewWorkflowTemplate.ID,
		ChangeMessage:      wtEvent.ChangeMessage,
		DataBefore:         wtEvent.OldWorkflowTemplate,
		DataAfter:          wtEvent.NewWorkflowTemplate,
	})
}

type addWorkflowTemplateInstanceAudit struct{}

func (a addWorkflowTemplateInstanceAudit) Compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error {
	var wtEvent sdk.EventWorkflowTemplateInstanceAdd
	if err := json.Unmarshal(e.Payload, &wtEvent); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal payload")
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
		},
		WorkflowTemplateInstanceID: wtEvent.WorkflowTemplateInstance.ID,
		DataType:                   "json",
		DataAfter:                  string(b),
	})
}

type updateWorkflowTemplateInstanceAudit struct{}

func (a updateWorkflowTemplateInstanceAudit) Compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error {
	var wtEvent sdk.EventWorkflowTemplateInstanceUpdate
	if err := json.Unmarshal(e.Payload, &wtEvent); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal payload")
	}

	before, err := json.Marshal(wtEvent.OldWorkflowTemplateInstance)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal workflow template instance")
	}

	after, err := json.Marshal(wtEvent.NewWorkflowTemplateInstance)
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal workflow template instance")
	}

	return InsertInstanceAudit(db, &sdk.AuditWorkflowTemplateInstance{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
		},
		WorkflowTemplateInstanceID: wtEvent.NewWorkflowTemplateInstance.ID,
		DataType:                   "json",
		DataBefore:                 string(before),
		DataAfter:                  string(after),
	})
}
