package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// DEPRECATED - Only use on old workflow
type NodeRunContext struct {
	CDS  CDSContext        `json:"cds,omitempty"`
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
	Event                map[string]interface{} `json:"event,omitempty"`
	Version              string                 `json:"version,omitempty"`
	ProjectKey           string                 `json:"project_key,omitempty"`
	RunID                string                 `json:"run_id,omitempty"`
	RunNumber            int64                  `json:"run_number,omitempty"`
	RunAttempt           int64                  `json:"run_attempt,omitempty"`
	Workflow             string                 `json:"workflow,omitempty"`
	WorkflowRef          string                 `json:"workflow_ref,omitempty"`
	WorkflowSha          string                 `json:"workflow_sha,omitempty"`
	WorkflowVCSServer    string                 `json:"workflow_vcs_server,omitempty"`
	WorkflowRepository   string                 `json:"workflow_repository,omitempty"`
	WorkflowIntegrations map[string]interface{} `json:"integrations,omitempty"` // actual key: artifact_manager
	TriggeringActor      string                 `json:"triggering_actor,omitempty"`

	// Job
	Job   string `json:"job,omitempty"`
	Stage string `json:"stage,omitempty"`
	// Worker
	Workspace         string `json:"workspace,omitempty"`
	ActionRef         string `json:"action_ref,omitempty"`
	ActionRespository string `json:"action_repository,omitempty"`
	ActionStatus      string `json:"action_status,omitempty"`
}

type GitContext struct {
	Hash       string `json:"hash,omitempty"`
	HashShort  string `json:"hash_short,omitempty"`
	Repository string `json:"repository,omitempty"`
	Branch     string `json:"branch,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Author     string `json:"author,omitempty"`
	Message    string `json:"message,omitempty"`
	URL        string `json:"url,omitempty"`
	Server     string `json:"server,omitempty"`
	EventName  string `json:"event_name,omitempty"`
	Connection string `json:"connection,omitempty"`
	SSHKey     string `json:"ssh_key,omitempty"`
	PGPKey     string `json:"pgp_key,omitempty"`
	HttpUser   string `json:"http_user,omitempty"`
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

type JobResultContext struct {
	Result  string          `json:"result"`
	Outputs JobResultOutput `json:"outputs"`
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
