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

// EventWorkflowTemplateInstanceAdd represents the event when adding a workflow template instance.
type EventWorkflowTemplateInstanceAdd struct {
	WorkflowTemplateInstance WorkflowTemplateInstance `json:"workflow_template_instance"`
}

// EventWorkflowTemplateInstanceUpdate represents the event when updating a workflow template instance.
type EventWorkflowTemplateInstanceUpdate struct {
	NewWorkflowTemplateInstance WorkflowTemplateInstance `json:"new_workflow_template_instance"`
	OldWorkflowTemplateInstance WorkflowTemplateInstance `json:"old_workflow_template_instance"`
}
