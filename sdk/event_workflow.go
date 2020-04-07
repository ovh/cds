package sdk

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
