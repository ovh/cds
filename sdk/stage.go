package sdk

import (
	"strings"
	"time"
)

// Stage Pipeline step that parallelize actions by order
type Stage struct {
	ID            int64                  `json:"id" yaml:"pipeline_stage_id"`
	Name          string                 `json:"name"`
	PipelineID    int64                  `json:"-" yaml:"-"`
	BuildOrder    int                    `json:"build_order"`
	Enabled       bool                   `json:"enabled"`
	RunJobs       []WorkflowNodeJobRun   `json:"run_jobs"`
	Prerequisites []Prerequisite         `json:"prerequisites"` //TODO: to delete
	Conditions    WorkflowNodeConditions `json:"conditions"`
	LastModified  time.Time              `json:"last_modified"`
	Jobs          []Job                  `json:"jobs"`
	Status        string                 `json:"status"`
	Warnings      []PipelineBuildWarning `json:"warnings"`
}

func (s *Stage) UnmarshalJSON(data []byte) error {
	var tmp struct {
		ID            int64                  `json:"id" yaml:"pipeline_stage_id"`
		Name          string                 `json:"name"`
		PipelineID    int64                  `json:"-" yaml:"-"`
		BuildOrder    int                    `json:"build_order"`
		Enabled       bool                   `json:"enabled"`
		RunJobs       []WorkflowNodeJobRun   `json:"run_jobs"`
		Prerequisites []Prerequisite         `json:"prerequisites"`
		Conditions    WorkflowNodeConditions `json:"conditions"`
		Jobs          []Job                  `json:"jobs"`
		Status        string                 `json:"status"`
		Warnings      []PipelineBuildWarning `json:"warnings"`
	}

	if err := JSONUnmarshal(data, &tmp); err != nil {
		return err
	}
	s.ID = tmp.ID
	s.Name = tmp.Name
	s.PipelineID = tmp.PipelineID
	s.BuildOrder = tmp.BuildOrder
	s.Enabled = tmp.Enabled
	s.RunJobs = tmp.RunJobs
	s.Prerequisites = tmp.Prerequisites
	s.Conditions = tmp.Conditions
	s.Jobs = tmp.Jobs
	s.Status = tmp.Status
	s.Warnings = tmp.Warnings

	var v map[string]interface{}
	if err := JSONUnmarshal(data, &v); err != nil {
		return err
	}
	if lastModifiedNumber, ok := v["last_modified"].(float64); ok {
		s.LastModified = time.Unix(int64(lastModifiedNumber), 0)
	}
	if lastModifiedString, ok := v["last_modified"].(string); ok {
		date, _ := time.Parse(time.RFC3339, lastModifiedString)
		s.LastModified = date
	}
	return nil
}

// StageSummary is a light representation of stage for CDS event
type StageSummary struct {
	ID             int64                       `json:"id"`
	Name           string                      `json:"name"`
	BuildOrder     int                         `json:"build_order"`
	Enabled        bool                        `json:"enabled"`
	Status         string                      `json:"status"`
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
func (s *Stage) PlainConditions() []WorkflowNodeCondition {
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

// NewStage instantiate a new Stage
func NewStage(name string) *Stage {
	s := &Stage{
		Name: name,
	}
	return s
}
