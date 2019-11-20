package event

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

func publishWorkflowTemplateEvent(ctx context.Context, payload interface{}, u sdk.Identifiable) {
	event := sdk.Event{
		Timestamp: time.Now(),
		Hostname:  hostname,
		CDSName:   cdsname,
		EventType: fmt.Sprintf("%T", payload),
		Payload:   structs.Map(payload),
	}
	if u != nil {
		event.Username = u.GetUsername()
		event.UserMail = u.GetEmail()
	}
	publishEvent(ctx, event)
}

// PublishWorkflowTemplateAdd publishes an event for the creation of the given workflow template.
func PublishWorkflowTemplateAdd(ctx context.Context, wt sdk.WorkflowTemplate, u sdk.Identifiable) {
	publishWorkflowTemplateEvent(ctx, sdk.EventWorkflowTemplateAdd{WorkflowTemplate: wt}, u)
}

// PublishWorkflowTemplateUpdate publishes an event for the update of the given workflow template.
func PublishWorkflowTemplateUpdate(ctx context.Context, old, new sdk.WorkflowTemplate, changeMessage string, u sdk.Identifiable) {
	publishWorkflowTemplateEvent(ctx, sdk.EventWorkflowTemplateUpdate{
		OldWorkflowTemplate: old,
		NewWorkflowTemplate: new,
		ChangeMessage:       changeMessage,
	}, u)
}

// PublishWorkflowTemplateInstanceAdd publishes an event for the creation of the given workflow template instance.
func PublishWorkflowTemplateInstanceAdd(ctx context.Context, wti sdk.WorkflowTemplateInstance, u sdk.Identifiable) {
	publishWorkflowTemplateEvent(ctx, sdk.EventWorkflowTemplateInstanceAdd{WorkflowTemplateInstance: wti}, u)
}

// PublishWorkflowTemplateInstanceUpdate publishes an event for the update of the given workflow template instance.
func PublishWorkflowTemplateInstanceUpdate(ctx context.Context, old, new sdk.WorkflowTemplateInstance, u sdk.Identifiable) {
	publishWorkflowTemplateEvent(ctx, sdk.EventWorkflowTemplateInstanceUpdate{
		OldWorkflowTemplateInstance: old,
		NewWorkflowTemplateInstance: new,
	}, u)
}
