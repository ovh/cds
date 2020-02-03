package sdk

import "github.com/mitchellh/mapstructure"

// EventWorkflowAdd represents the event when adding a workflow
type EventWorkflowAdd struct {
	Workflow Workflow `json:"workflow"`
}

// EventWorkflowUpdate represents the event when updating a workflow
type EventWorkflowUpdate struct {
	NewWorkflow Workflow `json:"new_workflow"`
	OldWorkflow Workflow `json:"old_workflow"`
}

// EventWorkflowDelete represents the event when deleting a workflow
type EventWorkflowDelete struct {
	Workflow Workflow `json:"workflow"`
}

// EventWorkflowPermissionAdd represents the event when adding a workflow permission
type EventWorkflowPermissionAdd struct {
	WorkflowID int64           `json:"workflow_id"`
	Permission GroupPermission `json:"group_permission"`
}

// EventWorkflowPermissionUpdate represents the event when updating a workflow permission
type EventWorkflowPermissionUpdate struct {
	WorkflowID    int64           `json:"workflow_id"`
	OldPermission GroupPermission `json:"old_group_permission"`
	NewPermission GroupPermission `json:"new_group_permission"`
}

// EventWorkflowPermissionDelete represents the event when deleting a workflow permission
type EventWorkflowPermissionDelete struct {
	WorkflowID int64           `json:"workflow_id"`
	Permission GroupPermission `json:"group_permission"`
}

// ToEventWorkflowPermissionAdd get the payload as EventWorkflowPermissionAdd
func (e Event) ToEventWorkflowPermissionAdd() (EventWorkflowPermissionAdd, error) {
	var permEvent EventWorkflowPermissionAdd
	if err := mapstructure.Decode(e.Payload, &permEvent); err != nil {
		return permEvent, WrapError(err, "ToEventWorkflowPermissionAdd> Unable to decode EventWorkflowPermissionAdd")
	}
	return permEvent, nil
}

// ToEventWorkflowPermissionDelete get the payload as EventWorkflowPermissionDelete
func (e Event) ToEventWorkflowPermissionDelete() (EventWorkflowPermissionDelete, error) {
	var permEvent EventWorkflowPermissionDelete
	if err := mapstructure.Decode(e.Payload, &permEvent); err != nil {
		return permEvent, WrapError(err, "ToEventWorkflowPermissionDelete> Unable to decode EventWorkflowPermissionDelete")
	}
	return permEvent, nil
}
