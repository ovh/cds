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

// AuditWorklflow represents an audit data on a workflow
type AuditWorklflow struct {
	ID          int64     `json:"id" db:"id"`
	ProjectKey  string    `json:"project_key" db:"project_key"`
	WorkflowID  int64     `json:"workflow_id" db:"workflow_id"`
	TriggeredBy string    `json:"triggered_by" db:"triggered_by"`
	Created     time.Time `json:"created" db:"created"`
	DataBefore  string    `json:"data_before" db:"data_before"`
	DataAfter   string    `json:"data_after" db:"data_after"`
	EventType   string    `json:"event_type" db:"event_type"`
	DataType    string    `json:"data_type" db:"data_type"`
}

// Audit represents audit interface
type Audit interface {
	Compute(db gorp.SqlExecutor, e Event) error
}
