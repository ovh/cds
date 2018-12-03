package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

func publishWorkflowTemplateEvent(payload interface{}, u *sdk.User) {
	event := sdk.Event{
		Timestamp: time.Now(),
		Hostname:  hostname,
		CDSName:   cdsname,
		EventType: fmt.Sprintf("%T", payload),
		Payload:   structs.Map(payload),
	}
	if u != nil {
		event.Username = u.Username
		event.UserMail = u.Email
	}
	publishEvent(event)
}

// PublishWorkflowTemplateAdd publishes an event for the creation of the given workflow template.
func PublishWorkflowTemplateAdd(wt sdk.WorkflowTemplate, u *sdk.User) {
	publishWorkflowTemplateEvent(sdk.EventWorkflowTemplateAdd{WorkflowTemplate: wt}, u)
}

// PublishWorkflowTemplateUpdate publishes an event for the update of the given workflow template.
func PublishWorkflowTemplateUpdate(new, old sdk.WorkflowTemplate, u *sdk.User) {
	publishWorkflowTemplateEvent(sdk.EventWorkflowTemplateUpdate{
		NewWorkflowTemplate: new,
		OldWorkflowTemplate: old,
	}, u)
}

// PublishWorkflowTemplateInstanceAdd publishes an event for the creation of the given workflow template instance.
func PublishWorkflowTemplateInstanceAdd(wti sdk.WorkflowTemplateInstance, u *sdk.User) {
	publishWorkflowTemplateEvent(sdk.EventWorkflowTemplateInstanceAdd{WorkflowTemplateInstance: wti}, u)
}

// PublishWorkflowTemplateInstanceUpdate publishes an event for the update of the given workflow template instance.
func PublishWorkflowTemplateInstanceUpdate(new, old sdk.WorkflowTemplateInstance, u *sdk.User) {
	publishWorkflowTemplateEvent(sdk.EventWorkflowTemplateInstanceUpdate{
		NewWorkflowTemplateInstance: new,
		OldWorkflowTemplateInstance: old,
	}, u)
}
