package sdk

// EventPipelineAdd represents the event when adding a pipeline
//easyjson:json
type EventPipelineAdd struct {
	Pipeline
}

// EventPipelineUpdate represents the event when updating a pipeline
//easyjson:json
type EventPipelineUpdate struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

// EventPipelineDelete represents the event when deleting a pipeline
//easyjson:json
type EventPipelineDelete struct {
}

// EventPipelineParameterAdd represents the event when adding a pipeline parameter
//easyjson:json
type EventPipelineParameterAdd struct {
	Parameter Parameter `json:"parameter"`
}

// EventPipelineParameterUpdate represents the event when updating a pipeline parameter
//easyjson:json
type EventPipelineParameterUpdate struct {
	OldParameter Parameter `json:"old_parameter"`
	NewParameter Parameter `json:"new_parameter"`
}

// EventPipelineParameterDelete represents the event when deleting a pipeline parameter
//easyjson:json
type EventPipelineParameterDelete struct {
	Parameter Parameter `json:"parameter"`
}

// EventPipelinePermissionAdd represents the event when adding a pipeline permission
//easyjson:json
type EventPipelinePermissionAdd struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventPipelinePermissionUpdate represents the event when updating a pipeline permission
//easyjson:json
type EventPipelinePermissionUpdate struct {
	OldPermission GroupPermission `json:"old_group_permission"`
	NewPermission GroupPermission `json:"new_group_permission"`
}

// EventPipelinePermissionDelete represents the event when deleting a pipeline permission
//easyjson:json
type EventPipelinePermissionDelete struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventPipelineStageAdd represents the event when adding a stage
//easyjson:json
type EventPipelineStageAdd struct {
	Name         string         `json:"name"`
	BuildOrder   int            `json:"build_order"`
	Enabled      bool           `json:"enabled"`
	Prerequisite []Prerequisite `json:"prerequisite"`
}

// EventPipelineStageMove represent the event when moving a stage
//easyjson:json
type EventPipelineStageMove struct {
	StageName          string `json:"stage_name"`
	StageID            int64  `json:"stage_id"`
	OldStageBuildOrder int    `json:"old_build_order"`
	NewStageBuildOrder int    `json:"new_build_order"`
}

// EventPipelineStageUpdate represents the event when updating a stage
//easyjson:json
type EventPipelineStageUpdate struct {
	NewName         string         `json:"name"`
	NewBuildOrder   int            `json:"build_order"`
	NewPrerequisite []Prerequisite `json:"prerequisite"`
	OldName         string         `json:"old_name"`
	OldBuildOrder   int            `json:"old_build_order"`
	OldPrerequisite []Prerequisite `json:"old_prerequisite"`
	NewEnabled      bool           `json:"enabled"`
	OldEnabled      bool           `json:"old_enabled"`
}

// EventPipelineStageDelete represents the event when deleting a stage
//easyjson:json
type EventPipelineStageDelete struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	BuildOrder int    `json:"build_order"`
}

// EventPipelineJobAdd represents the event when adding a job
//easyjson:json
type EventPipelineJobAdd struct {
	StageID         int64  `json:"stage_id"`
	StageName       string `json:"stage_name"`
	StageBuildOrder int    `json:"stage_build_order"`
	Job             Job    `json:"job"`
}

// EventPipelineJobUpdate represents the event when updating a job
//easyjson:json
type EventPipelineJobUpdate struct {
	StageID         int64  `json:"stage_id"`
	StageName       string `json:"stage_name"`
	StageBuildOrder int    `json:"stage_build_order"`
	OldJob          Job    `json:"old_job"`
	NewJob          Job    `json:"new_job"`
}

// EventPipelineJobDelete represents the event when deleting a job
//easyjson:json
type EventPipelineJobDelete struct {
	StageID         int64  `json:"stage_id"`
	StageName       string `json:"stage_name"`
	StageBuildOrder int    `json:"stage_build_order"`
	JobName         string `json:"job_name"`
}
