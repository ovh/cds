package sdk

// EventWorkflowTemplateAdd represents the event when adding a workflow template.
type EventWorkflowTemplateAdd struct {
	WorkflowTemplate WorkflowTemplate `json:"workflow_template"`
}

// EventWorkflowTemplateUpdate represents the event when updating a workflow template.
type EventWorkflowTemplateUpdate struct {
	NewWorkflowTemplate WorkflowTemplate `json:"new_workflow_template"`
	OldWorkflowTemplate WorkflowTemplate `json:"old_workflow_template"`
}

// EventWorkflowTemplateDelete represents the event when deleting a workflow template.
type EventWorkflowTemplateDelete struct {
	WorkflowTemplate WorkflowTemplate `json:"workflow_template"`
}
