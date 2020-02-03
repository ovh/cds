package sdk

// EventWorkflowTemplateAdd represents the event when adding a workflow template.
type EventWorkflowTemplateAdd struct {
	WorkflowTemplate WorkflowTemplate `json:"workflow_template"`
}

// EventWorkflowTemplateUpdate represents the event when updating a workflow template.
type EventWorkflowTemplateUpdate struct {
	OldWorkflowTemplate WorkflowTemplate `json:"old_workflow_template"`
	NewWorkflowTemplate WorkflowTemplate `json:"new_workflow_template"`
	ChangeMessage       string           `json:"change_message"`
}

// EventWorkflowTemplateInstanceAdd represents the event when adding a workflow template instance.
type EventWorkflowTemplateInstanceAdd struct {
	WorkflowTemplateInstance WorkflowTemplateInstance `json:"workflow_template_instance"`
}

// EventWorkflowTemplateInstanceUpdate represents the event when updating a workflow template instance.
type EventWorkflowTemplateInstanceUpdate struct {
	OldWorkflowTemplateInstance WorkflowTemplateInstance `json:"old_workflow_template_instance"`
	NewWorkflowTemplateInstance WorkflowTemplateInstance `json:"new_workflow_template_instance"`
}
