package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// DEPRECATED - Only use on old workflow
type NodeRunContext struct {
	Git  GitContext        `json:"git,omitempty"`
	Vars map[string]string `json:"vars,omitempty"`
	Jobs JobsResultContext `json:"jobs,omitempty"`
}

func (m NodeRunContext) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	return j, WrapError(err, "cannot marshal RunContext")
}

func (m *NodeRunContext) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte]) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, m), "cannot unmarshal RunContext")
}

type JobRunContext struct {
	NodeRunContext
	Job     JobContext        `json:"job,omitempty"`
	Secrets map[string]string `json:"secrets,omitempty"`
}

func (m JobRunContext) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	return j, WrapError(err, "cannot marshal JobRunContext")
}

func (m *JobRunContext) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, m), "cannot unmarshal JobRunContext")
}

type ActionContext struct {
	Inputs map[string]interface{} `json:"inputs,omitempty"`
}

type CDSContext struct {
	// Workflow
	EventName          string                 `json:"event_name,omitempty"`
	Event              map[string]interface{} `json:"event,omitempty"`
	ProjectKey         string                 `json:"project_key,omitempty"`
	RunID              string                 `json:"run_id,omitempty"`
	RunNumber          int64                  `json:"run_number,omitempty"`
	RunAttempt         int64                  `json:"run_attempt,omitempty"`
	Workflow           string                 `json:"workflow,omitempty"`
	WorkflowRef        string                 `json:"workflow_ref,omitempty"`
	WorkflowSha        string                 `json:"workflow_sha,omitempty"`
	WorkflowVCSServer  string                 `json:"workflow_vcs_server,omitempty"`
	WorkflowRepository string                 `json:"workflow_repository,omitempty"`
	TriggeringActor    string                 `json:"triggering_actor,omitempty"`

	// Workflow Template
	WorkflowTemplate                 string            `json:"workflow_template,omitempty"`
	WorkflowTemplateRef              string            `json:"workflow_template_ref,omitempty"`
	WorkflowTemplateSha              string            `json:"workflow_template_sha,omitempty"`
	WorkflowTemplateVCSServer        string            `json:"workflow_template_vcs_server,omitempty"`
	WorkflowTemplateRepository       string            `json:"workflow_template_repository,omitempty"`
	WorkflowTemplateProjectKey       string            `json:"workflow_template_project_key,omitempty"`
	WorkflowTemplateParams           map[string]string `json:"workflow_template_params,omitempty"`
	WorkflowTemplateCommitWebURL     string            `json:"workflow_template_commit_web_url,omitempty"`
	WorkflowTemplateRefWebURL        string            `json:"workflow_template_ref_web_url,omitempty"`
	WorkflowTemplateRepositoryWebURL string            `json:"workflow_template_repository_web_url,omitempty"`

	// Job
	Job   string `json:"job,omitempty"`
	Stage string `json:"stage,omitempty"`

	// Worker
	Workspace string `json:"workspace,omitempty"`

	// TODO
	WorkflowIntegrations map[string]interface{} `json:"integrations,omitempty"` // actual key: artifact_manager
}

type GitContext struct {
	Server           string   `json:"server,omitempty"`
	Repository       string   `json:"repository,omitempty"`
	RepositoryURL    string   `json:"repositoryUrl,omitempty"`
	RepositoryWebURL string   `json:"repository_web_url,omitempty"`
	RefWebURL        string   `json:"ref_web_url,omitempty"`
	CommitWebURL     string   `json:"commit_web_url,omitempty"`
	Ref              string   `json:"ref,omitempty"`
	RefName          string   `json:"ref_name,omitempty"`
	Sha              string   `json:"sha,omitempty"`
	RefType          string   `json:"ref_type,omitempty"`
	Connection       string   `json:"connection,omitempty"`
	SSHKey           string   `json:"ssh_key,omitempty"`
	Username         string   `json:"username,omitempty"`
	Token            string   `json:"token,omitempty"`
	SemverCurrent    string   `json:"semver_current,omitempty"`
	SemverNext       string   `json:"semver_next,omitempty"`
	ChangeSets       []string `json:"changesets,omitempty"`
}

type JobContext struct {
	// Update by worker
	Status string `json:"status"`

	// Set by hatchery
	Services map[string]JobContextService `json:"services"`
}

type JobContextService struct {
	ID   string            `json:"id"`
	Port map[string]string `json:"ports"`
}

type JobsResultContext map[string]JobResultContext

type JobsGateContext map[string]GateInputs

type JobResultContext struct {
	Result  V2WorkflowRunJobStatus `json:"result"`
	Outputs JobResultOutput        `json:"outputs"`
}

type JobResultOutput map[string]string

func (jro JobResultOutput) Value() (driver.Value, error) {
	j, err := json.Marshal(jro)
	return j, WrapError(err, "cannot marshal JobResultOutput")
}

func (jro *JobResultOutput) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.(string)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .(string) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal([]byte(source), jro), "cannot unmarshal JobResultOutput")
}

type StepsContext map[string]StepContext
type StepContext struct {
	Conclusion V2WorkflowRunJobStatus `json:"conclusion"` // result of a step after 'continue-on-error'
	Outcome    V2WorkflowRunJobStatus `json:"outcome"`    // result of a step before 'continue-on-error'
	Outputs    JobResultOutput        `json:"outputs"`
}

type NeedsContext map[string]NeedContext
type NeedContext struct {
	Result  V2WorkflowRunJobStatus `json:"result"`
	Outputs JobResultOutput        `json:"outputs"`
}

func (sc StepsContext) Value() (driver.Value, error) {
	j, err := json.Marshal(sc)
	return j, WrapError(err, "cannot marshal StepsContext")
}

func (sc *StepsContext) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.(string)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .(string) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal([]byte(source), sc), "cannot unmarshal StepsContext")
}
