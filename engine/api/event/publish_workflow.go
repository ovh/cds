package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

// publishWorkflowEvent publish workflow event
func publishWorkflowEvent(payload interface{}, key, workflowName string, eventIntegrations []sdk.ProjectIntegration, u *sdk.User) {
	eventIntegrationsId := make([]int64, len(eventIntegrations))
	for i, eventIntegration := range eventIntegrations {
		eventIntegrationsId[i] = eventIntegration.ID
	}

	event := sdk.Event{
		Timestamp:           time.Now(),
		Hostname:            hostname,
		CDSName:             cdsname,
		EventType:           fmt.Sprintf("%T", payload),
		Payload:             structs.Map(payload),
		ProjectKey:          key,
		WorkflowName:        workflowName,
		EventIntegrationsID: eventIntegrationsId,
	}
	if u != nil {
		event.Username = u.Username
		event.UserMail = u.Email
	}
	publishEvent(event)
}

// PublishWorkflowAdd publishes an event for the creation of the given Workflow
func PublishWorkflowAdd(projKey string, w sdk.Workflow, u *sdk.User) {
	e := sdk.EventWorkflowAdd{
		Workflow: w,
	}
	publishWorkflowEvent(e, projKey, w.Name, w.EventIntegrations, u)
}

// PublishWorkflowUpdate publishes an event for the update of the given Workflow
func PublishWorkflowUpdate(projKey string, w sdk.Workflow, oldw sdk.Workflow, u *sdk.User) {
	e := sdk.EventWorkflowUpdate{
		NewWorkflow: w,
		OldWorkflow: oldw,
	}
	publishWorkflowEvent(e, projKey, w.Name, w.EventIntegrations, u)
}

// PublishWorkflowDelete publishes an event for the deletion of the given Workflow
func PublishWorkflowDelete(projKey string, w sdk.Workflow, u *sdk.User) {
	e := sdk.EventWorkflowDelete{
		Workflow: w,
	}
	publishWorkflowEvent(e, projKey, w.Name, w.EventIntegrations, u)
}

// PublishWorkflowPermissionAdd publishes an event when adding a permission on a workflow
func PublishWorkflowPermissionAdd(projKey string, w sdk.Workflow, gp sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventWorkflowPermissionAdd{
		WorkflowID: w.ID,
		Permission: gp,
	}
	publishWorkflowEvent(e, projKey, w.Name, w.EventIntegrations, u)
}

// PublishWorkflowPermissionUpdate publishes an event when updating a permission on a workflow
func PublishWorkflowPermissionUpdate(projKey string, w sdk.Workflow, gp sdk.GroupPermission, gpOld sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventWorkflowPermissionUpdate{
		WorkflowID:    w.ID,
		NewPermission: gp,
		OldPermission: gpOld,
	}
	publishWorkflowEvent(e, projKey, w.Name, w.EventIntegrations, u)
}

// PublishWorkflowPermissionDelete publishes an event when deleting a permission on a workflow
func PublishWorkflowPermissionDelete(projKey string, w sdk.Workflow, gp sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventWorkflowPermissionDelete{
		WorkflowID: w.ID,
		Permission: gp,
	}
	publishWorkflowEvent(e, projKey, w.Name, w.EventIntegrations, u)
}
