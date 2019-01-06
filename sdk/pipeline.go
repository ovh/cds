package sdk

import (
	"time"

	"github.com/ovh/venom"
)

// Pipeline represents the complete behavior of CDS for each projects
type Pipeline struct {
	ID                int64             `json:"id" yaml:"-"`
	Name              string            `json:"name" cli:"name,key"`
	Description       string            `json:"description" cli:"description"`
	Type              string            `json:"type"`
	ProjectKey        string            `json:"projectKey"`
	ProjectID         int64             `json:"-"`
	LastPipelineBuild *PipelineBuild    `json:"last_pipeline_build"`
	Stages            []Stage           `json:"stages"`
	GroupPermission   []GroupPermission `json:"groups,omitempty"`
	Parameter         []Parameter       `json:"parameters,omitempty"`
	Usage             *Usage            `json:"usage,omitempty"`
	Permission        int               `json:"permission"`
	LastModified      int64             `json:"last_modified" cli:"modified"`
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

// PipelineBuild Struct for history table
type PipelineBuild struct {
	ID          int64                  `json:"id"`
	BuildNumber int64                  `json:"build_number"`
	Version     int64                  `json:"version"`
	Parameters  []Parameter            `json:"parameters"`
	Status      Status                 `json:"status"`
	Warnings    []PipelineBuildWarning `json:"warnings"`
	Start       time.Time              `json:"start,omitempty"`
	Done        time.Time              `json:"done,omitempty"`
	Stages      []Stage                `json:"stages"`

	Pipeline    Pipeline    `json:"pipeline"`
	Application Application `json:"application"`
	Environment Environment `json:"environment"`

	Artifacts             []Artifact           `json:"artifacts,omitempty"`
	Tests                 *venom.Tests         `json:"tests,omitempty"`
	Commits               []VCSCommit          `json:"commits,omitempty"`
	Trigger               PipelineBuildTrigger `json:"trigger"`
	PreviousPipelineBuild *PipelineBuild       `json:"previous_pipeline_build"`
}

// PipelineBuildTrigger Struct for history table
type PipelineBuildTrigger struct {
	ScheduledTrigger    bool           `json:"scheduled_trigger"`
	ManualTrigger       bool           `json:"manual_trigger"`
	TriggeredBy         *User          `json:"triggered_by"`
	ParentPipelineBuild *PipelineBuild `json:"parent_pipeline_build"`
	VCSChangesBranch    string         `json:"vcs_branch"`
	VCSChangesHash      string         `json:"vcs_hash"`
	VCSChangesAuthor    string         `json:"vcs_author"`
	VCSRemote           string         `json:"vcs_remote,omitempty"`
	VCSRemoteURL        string         `json:"vcs_remote_url,omitempty"`
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

// CDPipeline  Represent a pipeline in the CDTree
type CDPipeline struct {
	Project      Project         `json:"project"`
	Application  Application     `json:"application"`
	Environment  Environment     `json:"environment"`
	Pipeline     Pipeline        `json:"pipeline"`
	SubPipelines []CDPipeline    `json:"subPipelines"`
	Trigger      PipelineTrigger `json:"trigger"`
}

// Translate translates messages in pipelineBuild
func (p *PipelineBuild) Translate(lang string) {
	for ks := range p.Stages {
		for kj := range p.Stages[ks].PipelineBuildJobs {
			p.Stages[ks].PipelineBuildJobs[kj].Translate(lang)
		}
	}
}
