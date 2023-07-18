package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rockbears/yaml"
)

type V2WorkflowRun struct {
	ID           string             `json:"id" db:"id"`
	ProjectKey   string             `json:"project_key" db:"project_key"`
	VCSServerID  string             `json:"vcs_server_id" db:"vcs_server_id"`
	RepositoryID string             `json:"repository_id" db:"repository_id"`
	WorkflowName string             `json:"workflow_name" db:"workflow_name" cli:"workflow_name"`
	WorkflowSha  string             `json:"workflow_sha" db:"workflow_sha"`
	WorkflowRef  string             `json:"workflow_ref" db:"workflow_ref"`
	Status       string             `json:"status" db:"status" cli:"status"`
	RunNumber    int64              `json:"run_number" db:"run_number"`
	RunAttempt   int64              `json:"run_attempt" db:"run_attempt"`
	Started      time.Time          `json:"started" db:"started" cli:"started"`
	LastModified time.Time          `json:"last_modified" db:"last_modified" cli:"last_modified"`
	ToDelete     bool               `json:"to_delete" db:"to_delete"`
	WorkflowData V2WorkflowRunData  `json:"workflow_data" db:"workflow_data"`
	UserID       string             `json:"user_id" db:"user_id"`
	Username     string             `json:"username" db:"username" cli:"username"`
	Contexts     WorkflowRunContext `json:"contexts" db:"contexts"`
	Event        V2WorkflowRunEvent `json:"event" db:"event"`
}

type WorkflowRunContext struct {
	CDS  CDSContext        `json:"cds,omitempty"`
	Git  GitContext        `json:"git,omitempty"`
	Vars map[string]string `json:"vars,omitempty"`
}

type WorkflowRunJobsContext struct {
	WorkflowRunContext
	Jobs JobsResultContext `json:"jobs,omitempty"`
}

func (m WorkflowRunContext) Value() (driver.Value, error) {
	j, err := yaml.Marshal(m)
	return j, WrapError(err, "cannot marshal WorkflowRunContext")
}

func (m *WorkflowRunContext) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.(string)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .(string) failed (%T)", src))
	}
	return WrapError(yaml.Unmarshal([]byte(source), m), "cannot unmarshal WorkflowRunContext")
}

type V2WorkflowRunData struct {
	Workflow     V2Workflow               `json:"workflow"`
	WorkerModels map[string]V2WorkerModel `json:"worker_models"`
	Actions      map[string]V2Action      `json:"actions"`
}

func (w V2WorkflowRunData) Value() (driver.Value, error) {
	j, err := yaml.Marshal(w)
	return j, WrapError(err, "cannot marshal V2WorkflowRunData")
}

func (w *V2WorkflowRunData) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.(string)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .(string) failed (%T)", src))
	}
	return WrapError(yaml.Unmarshal([]byte(source), w), "cannot unmarshal V2WorkflowRunData")
}

type V2WorkflowRunEvent struct {
	Manual *ManualTrigger `json:"manual"`

	// TODO
	Scheduler      *SchedulerTrigger `json:"scheduler"`
	GitTrigger     *GitTrigger       `json:"git"`
	WebHookTrigger *WebHookTrigger   `json:"webhook"`
}

func (w V2WorkflowRunEvent) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal V2WorkflowRunTrigger")
}

func (w *V2WorkflowRunEvent) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, w), "cannot unmarshal V2WorkflowRunTrigger")
}

type ManualTrigger struct {
	Payload map[string]interface{} `json:"payload"`
}

type SchedulerTrigger struct {
	Payload map[string]interface{} `json:"payload"`
	Cron    string                 `json:"cron"`
}

type GitTrigger struct {
	EventName string                 `json:"event_name"`
	Payload   map[string]interface{} `json:"payload"`
}

type WebHookTrigger struct {
	Payload map[string]interface{} `json:"payload"`
}

type V2WorkflowRunJob struct {
	ID            string          `json:"id" db:"id"`
	WorkflowRunID string          `json:"workflow_run_id" db:"workflow_run_id"`
	ProjectKey    string          `json:"project_key" db:"project_key"`
	WorkflowName  string          `json:"workflow_name" db:"workflow_name"`
	RunNumber     int64           `json:"run_number" db:"run_number"`
	RunAttempt    int64           `json:"run_attempt" db:"run_attempt"`
	Status        string          `json:"status" db:"status"`
	Queued        time.Time       `json:"queued" db:"queued"`
	Started       time.Time       `json:"started" db:"started"`
	Ended         time.Time       `json:"ended" db:"ended"`
	JobID         string          `json:"job_id" db:"job_id"`
	Job           V2Job           `json:"job" db:"job"`
	WorkerID      string          `json:"worker_id,omitempty" db:"worker_id"`
	WorkerName    string          `json:"worker_name" db:"worker_name"`
	HatcheryName  string          `json:"hatchery_name" db:"hatchery_name"`
	Outputs       JobResultOutput `json:"outputs" db:"outputs"`
	UserID        string          `json:"user_id" db:"user_id"`
	Username      string          `json:"username" db:"username"`
	Region        string          `json:"region,omitempty" db:"region"`
	ModelType     string          `json:"model_type,omitempty" db:"model_type"`
}

type V2WorkflowRunEnqueue struct {
	RunID  string   `json:"run_id"`
	Jobs   []string `json:"jobs"`
	UserID string   `json:"user_id"`
}

type V2WorkflowRunInfo struct {
	ID            string    `json:"id" db:"id"`
	WorkflowRunID string    `json:"workflow_run_id" db:"workflow_run_id"`
	IssuedAt      time.Time `json:"issued_at" db:"issued_at"`
	Level         string    `json:"level" db:"level"`
	Message       string    `json:"message" db:"message"`
}

type V2WorkflowRunJobInfo struct {
	ID               string    `json:"id" db:"id"`
	WorkflowRunID    string    `json:"workflow_run_id" db:"workflow_run_id"`
	WorkflowRunJobID string    `json:"workflow_run_job_id" db:"workflow_run_job_id"`
	IssuedAt         time.Time `json:"issued_at" db:"issued_at"`
	Level            string    `json:"level" db:"level"`
	Message          string    `json:"message" db:"message"`
}

const (
	WorkflowRunInfoLevelInfo    = "info"
	WorkflowRunInfoLevelWarning = "warning"
	WorkflowRunInfoLevelError   = "error"
)

type V2WorkflowRunJobResult struct {
	Status string    `json:"status"`
	Error  string    `json:"error,omitempty"`
	Time   time.Time `json:"time"`
}

type V2SendJobRunInfo struct {
	Level   string    `json:"level" db:"level"`
	Message string    `json:"message" db:"message"`
	Time    time.Time `json:"time" db:"time"`
}
