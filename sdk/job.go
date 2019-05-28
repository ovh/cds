package sdk

// Job is the element of a stage
type Job struct {
	PipelineActionID int64                  `json:"pipeline_action_id"`
	PipelineStageID  int64                  `json:"pipeline_stage_id"`
	Enabled          bool                   `json:"enabled"`
	LastModified     int64                  `json:"last_modified"`
	Action           Action                 `json:"action"`
	Warnings         []PipelineBuildWarning `json:"warnings"`
}

// IsValid returns job's validity.
func (j Job) IsValid() error {
	if j.PipelineStageID == 0 {
		return NewErrorFrom(ErrWrongRequest, "invalid given stage id")
	}

	return j.Action.IsValid()
}
