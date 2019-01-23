package sdk

import (
	"time"
)

// Pipeline represents the complete behavior of CDS for each projects
type Pipeline struct {
	ID              int64             `json:"id" yaml:"-"`
	Name            string            `json:"name" cli:"name,key"`
	Description     string            `json:"description" cli:"description"`
	Type            string            `json:"type"`
	ProjectKey      string            `json:"projectKey"`
	ProjectID       int64             `json:"-"`
	Stages          []Stage           `json:"stages"`
	GroupPermission []GroupPermission `json:"groups,omitempty"`
	Parameter       []Parameter       `json:"parameters,omitempty"`
	Usage           *Usage            `json:"usage,omitempty"`
	Permission      int               `json:"permission"`
	LastModified    int64             `json:"last_modified" cli:"modified"`
}

// PipelineAudit represents pipeline audit
type PipelineAudit struct {
	ID         int64     `json:"id" db:"id"`
	PipelineID int64     `json:"pipeline_id" db:"pipeline_id"`
	UserName   string    `json:"username" db:"username"`
	Versionned time.Time `json:"versionned" db:"versionned"`
	Pipeline   *Pipeline `json:"pipeline" db:"-"`
	Action     string    `json:"action" db:"action"`
}

// PipelineBuildWarning Struct for display warnings about build
type PipelineBuildWarning struct {
	Type   string `json:"type"`
	Action Action `json:"action"`
}

// This constant deals with pipelines
const (
	// Different types of Pipeline
	BuildPipeline      = "build"      // DEPRECATED with workflows
	DeploymentPipeline = "deployment" // DEPRECATED with workflows
	TestingPipeline    = "testing"    // DEPRECATED with workflows
	// Different types of warning for PipelineBuild
	OptionalStepFailed = "optional_step_failed"
)

// AvailablePipelineType List of all pipeline type
var AvailablePipelineType = []string{
	BuildPipeline,
	DeploymentPipeline,
	TestingPipeline,
}

// PipelineAction represents an action in a pipeline
type PipelineAction struct {
	ActionName      string      `json:"actionName"`
	Args            []Parameter `json:"args"`
	PipelineStageID int64       `json:"pipeline_stage_id"`
}
