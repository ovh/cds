package workflowv3

import "github.com/ovh/cds/sdk"

func NewWorkflowRun() WorkflowRun {
	return WorkflowRun{
		JobRuns: make(map[string][]JobRun),
	}
}

type WorkflowRun struct {
	Number    int64                `json:"number,omitempty" yaml:"number,omitempty"`
	Infos     sdk.WorkflowRunInfos `json:"infos,omitempty" yaml:"infos,omitempty"`
	Status    string               `json:"status,omitempty" yaml:"status,omitempty"`
	JobRuns   map[string][]JobRun  `json:"job_runs,omitempty" yaml:"job_runs,omitempty"`
	Resources WorkflowRunResources `json:"resources,omitempty" yaml:"resources,omitempty"`
}

type WorkflowRunResources struct {
	Workflow     Workflow                 `json:"workflow,omitempty" yaml:"workflow,omitempty"`
	Integrations []sdk.ProjectIntegration `json:"integrations,omitempty" yaml:"integrations,omitempty"`
}
