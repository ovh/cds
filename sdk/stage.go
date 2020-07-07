package sdk

import "time"

// Stage Pipeline step that parallelize actions by order
type Stage struct {
	ID           int64                  `json:"id" yaml:"pipeline_stage_id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	PipelineID   int64                  `json:"-" yaml:"-" db:"pipeline_id"`
	BuildOrder   int                    `json:"build_order" db:"build_order"`
	Enabled      bool                   `json:"enabled"  db:"enabled"`
	Conditions   WorkflowNodeConditions `json:"conditions" db:"conditions"`
	LastModified time.Time              `json:"last_modified" db:"last_modified"`

	RunJobs []WorkflowNodeJobRun `json:"run_jobs" db:"-"`
	Jobs    []Job                `json:"jobs" db:"-"`
	Status  string               `json:"status" db:"-"`
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

// NewStage instantiate a new Stage
func NewStage(name string) *Stage {
	s := &Stage{
		Name: name,
	}
	return s
}
