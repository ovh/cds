package sdk

// Job is the element of a stage
type Job struct {
	PipelineActionID int64  `json:"pipeline_action_id"`
	PipelineStageID  int64  `json:"pipeline_stage_id"`
	Enabled          bool   `json:"enabled"`
	LastModified     int64  `json:"last_modified"`
	Action           Action `json:"action"`
}
