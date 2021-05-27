package workflowv3

import "time"

type JobRun struct {
	Status     string       `json:"status,omitempty" yaml:"status,omitempty"`
	SubNumber  int64        `json:"sub_number,omitempty" yaml:"sub_number,omitempty"`
	StepStatus []StepStatus `json:"step_status,omitempty" yaml:"step_status,omitempty"`
	// Info from workflow v2 model
	WorkflowNodeRunID    int64 `json:"workflow_node_run_id,omitempty" yaml:"workflow_node_run_id,omitempty"`
	WorkflowNodeJobRunID int64 `json:"workflow_node_job_run_id,omitempty" yaml:"workflow_node_job_run_id,omitempty"`
}

type StepStatus struct {
	StepOrder int64     `json:"step_order" yaml:"step_order"`
	Status    string    `json:"status,omitempty" yaml:"status,omitempty"`
	Start     time.Time `json:"start,omitempty" yaml:"start,omitempty"`
	Done      time.Time `json:"done,omitempty" yaml:"done,omitempty"`
}
