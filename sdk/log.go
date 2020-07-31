package sdk

import (
	"time"
)

// NewLog returns a log struct
func NewLog(JobID, NodeRunID int64, value string, stepOrder int) *Log {
	//There cant be any error since we are using time.Now which is obviously a real and valid timestamp
	now := time.Now()
	l := &Log{
		JobID:        JobID,
		NodeRunID:    NodeRunID,
		Start:        &now,
		StepOrder:    int64(stepOrder),
		Val:          value,
		LastModified: &now,
	}

	return l
}

type Log struct {
	ID           int64      `json:"id,omitempty" db:"id"`
	JobID        int64      `json:"workflow_node_run_job_id,omitempty" db:"workflow_node_run_job_id"`
	NodeRunID    int64      `json:"workflow_node_run_id,omitempty" db:"workflow_node_run_id"`
	Start        *time.Time `json:"start,omitempty" db:"start"`
	LastModified *time.Time `json:"lastModified,omitempty" db:"last_modified"`
	Done         *time.Time `json:"done,omitempty" db:"done"`
	StepOrder    int64      `json:"stepOrder,omitempty" db:"step_order"`
	Val          string     `json:"val,omitempty" db:"value"`
}

type ServiceLog struct {
	ID                     int64      `json:"id,omitempty" db:"id"`
	WorkflowNodeJobRunID   int64      `json:"workflow_node_run_job_id" db:"workflow_node_run_job_id"`
	WorkflowNodeRunID      int64      `json:"workflow_node_run_id" db:"workflow_node_run_id"`
	Start                  *time.Time `json:"start" db:"start"`
	LastModified           *time.Time `json:"last_modified" db:"last_modified"`
	ServiceRequirementID   int64      `json:"requirement_id" db:"-"`
	ServiceRequirementName string     `json:"requirement_service_name" db:"requirement_service_name"`
	Val                    string     `json:"val,omitempty" db:"value"`

	// aggregate
	ProjectKey   string `json:"project_key" db:"-"`
	WorkflowName string `json:"workflow_name" db:"-"`
	WorkflowID   int64  `json:"workflow_id" db:"-"`
	RunID        int64  `json:"run_id" db:"-"`
	NodeRunName  string `json:"node_run_name" db:"-"`
	JobName      string `json:"job_name" db:"-"`
	WorkerName   string `json:"worker_name" db:"-"`
}
