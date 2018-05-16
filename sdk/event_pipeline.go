package sdk

// EventPipelineAdd represents the event when adding a pipeline
type EventPipelineAdd struct {
	Pipeline
}

// EventPipelineUpdate represents the event when updating a pipeline
type EventPipelineUpdate struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

// EventPipelineDelete represents the event when deleting a pipeline
type EventPipelineDelete struct {
}

// EventPipelineParameterAdd represents the event when adding a pipeline parameter
type EventPipelineParameterAdd struct {
	Parameter Parameter `json:"parameter"`
}

// EventPipelineParameterUpdate represents the event when updating a pipeline parameter
type EventPipelineParameterUpdate struct {
	OldParameter Parameter `json:"old_parameter"`
	NewParameter Parameter `json:"new_parameter"`
}

// EventPipelineParameterDelete represents the event when deleting a pipeline parameter
type EventPipelineParameterDelete struct {
	Parameter Parameter `json:"parameter"`
}

// EventPipelinePermissionAdd represents the event when adding a pipeline permission
type EventPipelinePermissionAdd struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventPipelinePermissionUpdate represents the event when updating a pipeline permission
type EventPipelinePermissionUpdate struct {
	OldPermission GroupPermission `json:"old_group_permission"`
	NewPermission GroupPermission `json:"new_group_permission"`
}

// EventPipelinePermissionDelete represents the event when deleting a pipeline permission
type EventPipelinePermissionDelete struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventPipelineStageAdd represents the event when adding a stage
type EventPipelineStageAdd struct {
	Name         string         `json:"name"`
	BuildOrder   int            `json:"build_order"`
	Enabled      bool           `json:"enabled"`
	Prerequisite []Prerequisite `json:"prerequisite"`
}

// EventPipelineStageUpdate represents the event when updating a stage
type EventPipelineStageUpdate struct {
	NewName         string         `json:"name"`
	NewBuildOrder   int            `json:"build_order"`
	NewEnabled      bool           `json:"enabled"`
	NewPrerequisite []Prerequisite `json:"prerequisite"`
	OldName         string         `json:"name"`
	OldBuildOrder   int            `json:"build_order"`
	OldEnabled      bool           `json:"enabled"`
	OldPrerequisite []Prerequisite `json:"prerequisite"`
}

// EventPipelineStageDelete represents the event when deleting a stage
type EventPipelineStageDelete struct {
	Name         string         `json:"name"`
	BuildOrder   int            `json:"build_order"`
	Enabled      bool           `json:"enabled"`
	Prerequisite []Prerequisite `json:"prerequisite"`
}

// EventPipelineJobAdd represents the event when adding a job
type EventPipelineJobAdd struct {
	Job
}

// EventPipelineJobUpdate represents the event when updating a job
type EventPipelineJobUpdate struct {
	OldJob Job
	NewJob Job
}

// EventPipelineJobDelete represents the event when deleting a job
type EventPipelineJobDelete struct {
	Job
}
