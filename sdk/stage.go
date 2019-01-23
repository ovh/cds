package sdk

import (
	"strings"
)

// Stage Pipeline step that parallelize actions by order
type Stage struct {
	ID            int64                  `json:"id" yaml:"pipeline_stage_id"`
	Name          string                 `json:"name"`
	PipelineID    int64                  `json:"-" yaml:"-"`
	BuildOrder    int                    `json:"build_order"`
	Enabled       bool                   `json:"enabled"`
	RunJobs       []WorkflowNodeJobRun   `json:"run_jobs"`
	Prerequisites []Prerequisite         `json:"prerequisites"`
	LastModified  int64                  `json:"last_modified"`
	Jobs          []Job                  `json:"jobs"`
	Status        Status                 `json:"status"`
	Warnings      []PipelineBuildWarning `json:"warnings"`
}

// StageSummary is a light representation of stage for CDS event
type StageSummary struct {
	ID             int64                       `json:"id"`
	Name           string                      `json:"name"`
	BuildOrder     int                         `json:"build_order"`
	Enabled        bool                        `json:"enabled"`
	Status         Status                      `json:"status"`
	Jobs           []Job                       `json:"jobs"`
	RunJobsSummary []WorkflowNodeJobRunSummary `json:"run_jobs_summary"`
}

// ToSummary transforms a Stage into a StageSummary
func (s Stage) ToSummary() StageSummary {
	sum := StageSummary{
		ID:             s.ID,
		Name:           s.Name,
		BuildOrder:     s.BuildOrder,
		Enabled:        s.Enabled,
		Status:         s.Status,
		RunJobsSummary: make([]WorkflowNodeJobRunSummary, len(s.RunJobs)),
		Jobs:           s.Jobs,
	}
	for i := range s.RunJobs {
		sum.RunJobsSummary[i] = s.RunJobs[i].ToSummary()
	}
	return sum
}

// Conditions returns stage prerequisites as a set of WorkflowTriggerCondition regex
func (s *Stage) Conditions() []WorkflowNodeCondition {
	res := make([]WorkflowNodeCondition, len(s.Prerequisites))
	for i, p := range s.Prerequisites {
		if !strings.HasPrefix(p.Parameter, "workflow.") && !strings.HasPrefix(p.Parameter, "git.") {
			p.Parameter = "cds.pip." + p.Parameter
		}
		res[i] = WorkflowNodeCondition{
			Value:    p.ExpectedValue,
			Variable: p.Parameter,
			Operator: WorkflowConditionsOperatorRegex,
		}
	}
	return res
}

// NewStage instanciate a new Stage
func NewStage(name string) *Stage {
	s := &Stage{
		Name: name,
	}
	return s
}
