package event

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

// publishWorkflowEvent publish workflow event
func publishWorkflowEvent(ctx context.Context, payload interface{}, key, workflowName string, eventIntegrations []sdk.ProjectIntegration, u sdk.Identifiable) {
	eventIntegrationsID := make([]int64, len(eventIntegrations))
	for i, eventIntegration := range eventIntegrations {
		eventIntegrationsID[i] = eventIntegration.ID
	}

	event := sdk.Event{
		Timestamp:           time.Now(),
		Hostname:            hostname,
		CDSName:             cdsname,
		EventType:           fmt.Sprintf("%T", payload),
		Payload:             structs.Map(payload),
		ProjectKey:          key,
		WorkflowName:        workflowName,
		EventIntegrationsID: eventIntegrationsID,
	}
	if u != nil {
		event.Username = u.GetUsername()
		event.UserMail = u.GetEmail()
	}
	publishEvent(ctx, event)
}

// PublishWorkflowAdd publishes an event for the creation of the given Workflow
func PublishWorkflowAdd(ctx context.Context, projKey string, w sdk.Workflow, u sdk.Identifiable) {
	e := sdk.EventWorkflowAdd{
		Workflow: w,
	}
	publishWorkflowEvent(ctx, e, projKey, w.Name, w.EventIntegrations, u)
}

// PublishWorkflowUpdate publishes an event for the update of the given Workflow
func PublishWorkflowUpdate(ctx context.Context, projKey string, w sdk.Workflow, oldw sdk.Workflow, u sdk.Identifiable) {
	e := sdk.EventWorkflowUpdate{
		NewWorkflow: w,
		OldWorkflow: oldw,
	}
	publishWorkflowEvent(ctx, e, projKey, w.Name, w.EventIntegrations, u)
}

// PublishWorkflowDelete publishes an event for the deletion of the given Workflow
func PublishWorkflowDelete(ctx context.Context, projKey string, w sdk.Workflow, u sdk.Identifiable) {
	e := sdk.EventWorkflowDelete{
		Workflow: w,
	}
	publishWorkflowEvent(ctx, e, projKey, w.Name, w.EventIntegrations, u)
}

// PublishWorkflowPermissionAdd publishes an event when adding a permission on a workflow
func PublishWorkflowPermissionAdd(ctx context.Context, projKey string, w sdk.Workflow, gp sdk.GroupPermission, u sdk.Identifiable) {
	e := sdk.EventWorkflowPermissionAdd{
		WorkflowID: w.ID,
		Permission: gp,
	}
	publishWorkflowEvent(ctx, e, projKey, w.Name, w.EventIntegrations, u)
}

// PublishWorkflowPermissionUpdate publishes an event when updating a permission on a workflow
func PublishWorkflowPermissionUpdate(ctx context.Context, projKey string, w sdk.Workflow, gp sdk.GroupPermission, gpOld sdk.GroupPermission, u sdk.Identifiable) {
	e := sdk.EventWorkflowPermissionUpdate{
		WorkflowID:    w.ID,
		NewPermission: gp,
		OldPermission: gpOld,
	}
	publishWorkflowEvent(ctx, e, projKey, w.Name, w.EventIntegrations, u)
}

// PublishWorkflowPermissionDelete publishes an event when deleting a permission on a workflow
func PublishWorkflowPermissionDelete(ctx context.Context, projKey string, w sdk.Workflow, gp sdk.GroupPermission, u sdk.Identifiable) {
	e := sdk.EventWorkflowPermissionDelete{
		WorkflowID: w.ID,
		Permission: gp,
	}
	publishWorkflowEvent(ctx, e, projKey, w.Name, w.EventIntegrations, u)
}
