package sdk

import (
	"time"

	"github.com/go-gorp/gorp"
)

// Different type of Audit event
const (
	AuditAdd    = "add"
	AuditUpdate = "update"
	AuditDelete = "delete"
)

// AuditCommon contains basic stuff for audits.
type AuditCommon struct {
	ID          int64     `json:"id" db:"id"`
	TriggeredBy string    `json:"triggered_by" db:"triggered_by"`
	Created     time.Time `json:"created" db:"created" mapstructure:"-"`
	EventType   string    `json:"event_type" db:"event_type"`
}

// AuditWorkflow represents an audit data on a workflow.
type AuditWorkflow struct {
	AuditCommon
	ProjectKey string `json:"project_key" db:"project_key"`
	WorkflowID int64  `json:"workflow_id" db:"workflow_id"`
	DataType   string `json:"data_type" db:"data_type"`
	DataBefore string `json:"data_before" db:"data_before"`
	DataAfter  string `json:"data_after" db:"data_after"`
}

// Audit represents audit interface.
type Audit interface {
	Compute(db gorp.SqlExecutor, e Event) error
}

// AuditWorkflowTemplate represents an audit data on a workflow template.
type AuditWorkflowTemplate struct {
	AuditCommon
	WorkflowTemplateID int64            `json:"workflow_template_id" db:"workflow_template_id"`
	ChangeMessage      string           `json:"change_message,omitempty" db:"change_message"`
	DataBefore         WorkflowTemplate `json:"data_before" db:"data_before"`
	DataAfter          WorkflowTemplate `json:"data_after" db:"data_after"`
}

// AuditWorkflowTemplateInstance represents an audit data on a workflow template instance.
type AuditWorkflowTemplateInstance struct {
	AuditCommon
	WorkflowTemplateInstanceID int64  `json:"workflow_template_instance_id" db:"workflow_template_instance_id"`
	DataType                   string `json:"data_type" db:"data_type"`
	DataBefore                 string `json:"data_before" db:"data_before"`
	DataAfter                  string `json:"data_after" db:"data_after"`
}

// AuditAction represents an audit data on a action.
type AuditAction struct {
	AuditCommon
	ActionID   int64  `json:"action_id" db:"action_id"`
	DataType   string `json:"data_type" db:"data_type"`
	DataBefore string `json:"data_before" db:"data_before"`
	DataAfter  string `json:"data_after" db:"data_after"`
}
