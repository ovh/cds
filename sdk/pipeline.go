package sdk

import (
	"time"
)

// Pipeline represents the complete behavior of CDS for each projects
type Pipeline struct {
	ID             int64         `json:"id" yaml:"-" db:"id"`
	Name           string        `json:"name" cli:"name,key" db:"name" action_metadata:"pipeline-key"`
	Description    string        `json:"description" cli:"description" db:"description"`
	ProjectKey     string        `json:"projectKey" db:"projectKey"`
	ProjectID      int64         `json:"-" db:"project_id"`
	Stages         []Stage       `json:"stages"`
	Parameter      []Parameter   `json:"parameters,omitempty"`
	Usage          *Usage        `json:"usage,omitempty"`
	LastModified   int64         `json:"last_modified" cli:"modified"`
	FromRepository string        `json:"from_repository" cli:"from_repository" db:"from_repository"`
	AsCodeEvents   []AsCodeEvent `json:"ascode_events" cli:"-" db:"-"`
	// aggregate
	WorkflowAscodeHolder *Workflow `json:"workflow_ascode_holder,omitempty" cli:"-" db:"-"`
}

func (p Pipeline) IsValid() error {
	// check pipeline name pattern
	if regexp := NamePatternRegex; !regexp.MatchString(p.Name) {
		return NewErrorFrom(ErrInvalidPipelinePattern, "pipeline name %s do not respect pattern %s", p.Name, NamePattern)
	}
	return nil
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
	// Different types of warning for PipelineBuild
	OptionalStepFailed = "optional_step_failed"
)

// PipelineAction represents an action in a pipeline
type PipelineAction struct {
	ActionName      string      `json:"actionName"`
	Args            []Parameter `json:"args"`
	PipelineStageID int64       `json:"pipeline_stage_id"`
}
