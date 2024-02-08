package sdk

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/rockbears/yaml"
)

const (
	GitBranchManualPayload = "git.branch"
	GitCommitManualPayload = "git.commit"
	GitTagManualPayload    = "git.tag"
)

type V2WorkflowRunHookRequest struct {
	HookEventID   string                 `json:"hook_event_id"`
	UserID        string                 `json:"user_id"`
	EventName     string                 `json:"event_name"`
	Ref           string                 `json:"ref"`
	Sha           string                 `json:"sha"`
	Payload       map[string]interface{} `json:"payload"`
	HookType      string                 `json:"hook_type"`
	EntityUpdated string                 `json:"entity_updated"`
	SemverCurrent string                 `json:"semver_current"`
	SemverNext    string                 `json:"semver_next"`
}

type V2WorkflowRun struct {
	ID           string                 `json:"id" db:"id"`
	ProjectKey   string                 `json:"project_key" db:"project_key"`
	VCSServerID  string                 `json:"vcs_server_id" db:"vcs_server_id"`
	VCSServer    string                 `json:"vcs_server" db:"vcs_server"`
	RepositoryID string                 `json:"repository_id" db:"repository_id"`
	Repository   string                 `json:"repository" db:"repository"`
	WorkflowName string                 `json:"workflow_name" db:"workflow_name" cli:"workflow_name"`
	WorkflowSha  string                 `json:"workflow_sha" db:"workflow_sha"`
	WorkflowRef  string                 `json:"workflow_ref" db:"workflow_ref"`
	Status       string                 `json:"status" db:"status" cli:"status"`
	RunNumber    int64                  `json:"run_number" db:"run_number" cli:"run_number"`
	RunAttempt   int64                  `json:"run_attempt" db:"run_attempt"`
	Started      time.Time              `json:"started" db:"started" cli:"started"`
	LastModified time.Time              `json:"last_modified" db:"last_modified" cli:"last_modified"`
	ToDelete     bool                   `json:"to_delete" db:"to_delete"`
	WorkflowData V2WorkflowRunData      `json:"workflow_data" db:"workflow_data"`
	UserID       string                 `json:"user_id" db:"user_id"`
	Username     string                 `json:"username" db:"username" cli:"username"`
	Contexts     WorkflowRunContext     `json:"contexts" db:"contexts"`
	RunEvent     V2WorkflowRunEvent     `json:"event" db:"event"`
	RunJobEvent  V2WorkflowRunJobEvents `json:"job_events" db:"job_event"`

	// Aggregations
	Results []V2WorkflowRunResult `json:"results" db:"-"`
}

type WorkflowRunContext struct {
	CDS CDSContext        `json:"cds,omitempty"`
	Git GitContext        `json:"git,omitempty"`
	Env map[string]string `json:"env,omitempty"`
}

func (m WorkflowRunContext) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	return j, WrapError(err, "cannot marshal WorkflowRunContext")
}

func (m *WorkflowRunContext) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, m), "cannot unmarshal WorkflowRunContext")
}

type WorkflowRunJobsContext struct {
	WorkflowRunContext
	Jobs         JobsResultContext       `json:"jobs"`
	Needs        NeedsContext            `json:"needs"`
	Inputs       map[string]interface{}  `json:"inputs"`
	Steps        StepsContext            `json:"steps"`
	Matrix       map[string]string       `json:"matrix"`
	Integrations *JobIntegrationsContext `json:"integrations,omitempty"`
	Gate         map[string]interface{}  `json:"gate"`
	Vars         map[string]interface{}  `json:"vars"`
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

type V2WorkflowRunJobEvent struct {
	UserID     string                 `json:"user_id"`
	Username   string                 `json:"username"`
	JobID      string                 `json:"job_id"`
	Inputs     map[string]interface{} `json:"inputs"`
	RunAttempt int64                  `json:"run_attempt"`
}

type V2WorkflowRunJobEvents []V2WorkflowRunJobEvent

func (w V2WorkflowRunJobEvents) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal V2WorkflowRunJobEvents")
}

func (w *V2WorkflowRunJobEvents) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, w), "cannot unmarshal V2WorkflowRunJobEvents")
}

type V2WorkflowRunEvent struct {
	Manual                *ManualTrigger         `json:"manual,omitempty"`
	GitTrigger            *GitTrigger            `json:"git,omitempty"`
	WorkflowUpdateTrigger *WorkflowUpdateTrigger `json:"workflow_update,omitempty"`
	ModelUpdateTrigger    *ModelUpdateTrigger    `json:"model_update,omitempty"`

	// TODO
	Scheduler      *SchedulerTrigger `json:"scheduler"`
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
	EventName     string                 `json:"event_name"`
	Payload       map[string]interface{} `json:"payload"`
	Ref           string                 `json:"ref"`
	Sha           string                 `json:"sha"`
	SemverCurrent string                 `json:"semver_current"`
	SemverNext    string                 `json:"sember_next"`
}

type WorkflowUpdateTrigger struct {
	WorkflowUpdated string `json:"workflow_updated"`
	Ref             string `json:"ref"`
}

type ModelUpdateTrigger struct {
	ModelUpdated string `json:"model_updated"`
	Ref          string `json:"ref"`
}

type WebHookTrigger struct {
	Payload map[string]interface{} `json:"payload"`
}

type V2WorkflowRunJob struct {
	ID            string         `json:"id" db:"id"`
	JobID         string         `json:"job_id" db:"job_id" cli:"job_id"`
	WorkflowRunID string         `json:"workflow_run_id" db:"workflow_run_id"`
	ProjectKey    string         `json:"project_key" db:"project_key"`
	WorkflowName  string         `json:"workflow_name" db:"workflow_name"`
	RunNumber     int64          `json:"run_number" db:"run_number"`
	RunAttempt    int64          `json:"run_attempt" db:"run_attempt"`
	Status        string         `json:"status" db:"status" cli:"status"`
	Queued        time.Time      `json:"queued" db:"queued"`
	Scheduled     time.Time      `json:"scheduled" db:"scheduled"`
	Started       time.Time      `json:"started" db:"started"`
	Ended         time.Time      `json:"ended" db:"ended"`
	Job           V2Job          `json:"job" db:"job"`
	WorkerID      string         `json:"worker_id,omitempty" db:"worker_id"`
	WorkerName    string         `json:"worker_name" db:"worker_name"`
	HatcheryName  string         `json:"hatchery_name" db:"hatchery_name"`
	StepsStatus   JobStepsStatus `json:"steps_status" db:"steps_status"`
	UserID        string         `json:"user_id" db:"user_id"`
	Username      string         `json:"username" db:"username"`
	Region        string         `json:"region,omitempty" db:"region"`
	ModelType     string         `json:"model_type,omitempty" db:"model_type"`
	Matrix        JobMatrix      `json:"matrix,omitempty" db:"matrix"`
	GateInputs    GateInputs     `json:"gate_inputs,omitempty" db:"gate_inputs"`
}

type JobIntegrationsContext struct {
	ArtifactManager string `json:"artifact_manager,omitempty"`
	Deployment      string `json:"deployment,omitempty"`
}

func (c JobIntegrationsContext) All() []string {
	var res []string
	if c.ArtifactManager != "" {
		res = append(res, c.ArtifactManager)
	}
	if c.Deployment != "" {
		res = append(res, c.Deployment)
	}
	return res
}

type JobStepsStatus map[string]JobStepStatus
type JobStepStatus struct {
	Conclusion string          `json:"conclusion"` // result of a step after 'continue-on-error'
	Outcome    string          `json:"outcome"`    // result of a step before 'continue-on-error'
	Outputs    JobResultOutput `json:"outputs"`
	Started    time.Time       `json:"started"`
	Ended      time.Time       `json:"ended"`
}

type GateInputs map[string]interface{}

func (gi GateInputs) Value() (driver.Value, error) {
	m, err := yaml.Marshal(gi)
	return m, WrapError(err, "cannot marshal GateInputs")
}

func (gi *GateInputs) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.(string)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .(string) failed (%T)", src))
	}
	return WrapError(yaml.Unmarshal([]byte(source), gi), "cannot unmarshal GateInputs")
}

type JobMatrix map[string]string

func (jm JobMatrix) Value() (driver.Value, error) {
	m, err := yaml.Marshal(jm)
	return m, WrapError(err, "cannot marshal JobMatrix")
}

func (jm *JobMatrix) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.(string)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .(string) failed (%T)", src))
	}
	return WrapError(yaml.Unmarshal([]byte(source), jm), "cannot unmarshal JobMatrix")
}

func (sc JobStepsStatus) Value() (driver.Value, error) {
	j, err := json.Marshal(sc)
	return j, WrapError(err, "cannot marshal JobStepsStatus")
}

func (sc *JobStepsStatus) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.(string)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .(string) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal([]byte(source), sc), "cannot unmarshal JobStepsStatus")
}

func (s JobStepsStatus) ToStepContext() StepsContext {
	stepsContext := StepsContext{}
	for k, v := range s {
		// Do not include current step
		if v.Conclusion == "" {
			continue
		}
		stepsContext[k] = StepContext{
			Conclusion: v.Conclusion,
			Outcome:    v.Outcome,
			Outputs:    v.Outputs,
		}
	}
	return stepsContext
}

type V2WorkflowRunEnqueue struct {
	RunID  string                   `json:"run_id"`
	UserID string                   `json:"user_id"`
	Gate   V2WorkflowRunEnqueueGate `json:"gate"`
}

type V2WorkflowRunEnqueueGate struct {
	JobID  string                 `json:"job_id"`
	Inputs map[string]interface{} `json:"inputs"`
}

type V2WorkflowRunInfo struct {
	ID            string    `json:"id" db:"id"`
	WorkflowRunID string    `json:"workflow_run_id" db:"workflow_run_id"`
	IssuedAt      time.Time `json:"issued_at" db:"issued_at" cli:"issue_at"`
	Level         string    `json:"level" db:"level" cli:"level"`
	Message       string    `json:"message" db:"message" cli:"message"`
}

type V2WorkflowRunJobInfo struct {
	ID               string    `json:"id" db:"id"`
	WorkflowRunID    string    `json:"workflow_run_id" db:"workflow_run_id"`
	WorkflowRunJobID string    `json:"workflow_run_job_id" db:"workflow_run_job_id"`
	IssuedAt         time.Time `json:"issued_at" db:"issued_at" cli:"date"`
	Level            string    `json:"level" db:"level" cli:"level"`
	Message          string    `json:"message" db:"message" cli:"message"`
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

func GetJobStepName(stepID string, stepIndex int) string {
	if stepID != "" {
		return stepID
	}
	return fmt.Sprintf("step-%d", stepIndex)

}

type WorkflowRunStages map[string]WorkflowRunStage

func (wrs WorkflowRunStages) ComputeStatus() {
	// Compute job status
stageLoop:
	for name := range wrs {
		stage := wrs[name]
		for _, status := range stage.Jobs {
			if !StatusIsTerminated(status) {
				stage.Ended = false
				wrs[name] = stage
				continue stageLoop
			}
		}
		stage.Ended = true
		wrs[name] = stage
	}

	// Compute stage needs
	for name := range wrs {
		stage := wrs[name]

		canBeRun := true
		for _, n := range stage.Needs {
			if !wrs[n].Ended {
				canBeRun = false
				break
			}
		}
		stage.CanBeRun = canBeRun
		wrs[name] = stage
	}
}

type WorkflowRunStage struct {
	WorkflowStage
	CanBeRun bool
	Jobs     map[string]string
	Ended    bool
}

func (w V2WorkflowRun) GetStages() WorkflowRunStages {
	stages := WorkflowRunStages{}
	for k, s := range w.WorkflowData.Workflow.Stages {
		stages[k] = WorkflowRunStage{WorkflowStage: s, Jobs: make(map[string]string)}
	}
	if len(stages) == 0 {
		return stages
	}
	for jobID, job := range w.WorkflowData.Workflow.Jobs {
		stages[job.Stage].Jobs[jobID] = ""
	}
	return stages
}

type V2WorkflowRunResult struct {
	ID                             string                                      `json:"id" db:"id"`
	WorkflowRunID                  string                                      `json:"workflow_run_id" db:"workflow_run_id"`
	WorkflowRunJobID               string                                      `json:"workflow_run_job_id" db:"workflow_run_job_id"`
	RunAttempt                     int64                                       `json:"run_attempt" db:"run_attempt"`
	IssuedAt                       time.Time                                   `json:"issued_at" db:"issued_at"`
	Type                           V2WorkflowRunResultType                     `json:"type" db:"type"`
	ArtifactManagerIntegrationName *string                                     `json:"artifact_manager_integration_name" db:"artifact_manager_integration_name"`
	ArtifactManagerMetadata        *V2WorkflowRunResultArtifactManagerMetadata `json:"artifact_manager_metadata" db:"artifact_manager_metadata"`
	Detail                         V2WorkflowRunResultDetail                   `json:"detail" db:"artifact_manager_detail"`
	DataSync                       *WorkflowRunResultSync                      `json:"sync,omitempty" db:"sync"`
	Status                         string                                      `json:"status" db:"status"`
}

func (r *V2WorkflowRunResult) GetDetail() (any, error) {
	if err := r.Detail.castData(); err != nil {
		return nil, err
	}
	return r.Detail.Data, nil
}

func (r *V2WorkflowRunResult) GetDetailAsV2WorkflowRunResultDockerDetail() (*V2WorkflowRunResultDockerDetail, error) {
	if err := r.Detail.castData(); err != nil {
		return nil, err
	}
	i, ok := r.Detail.Data.(*V2WorkflowRunResultDockerDetail)
	if !ok {
		var ii V2WorkflowRunResultDockerDetail
		ii, ok = r.Detail.Data.(V2WorkflowRunResultDockerDetail)
		if ok {
			i = &ii
		}
	}
	if !ok {
		return nil, errors.New("unable to cast detail as V2WorkflowRunResultDockerDetail")
	}
	return i, nil
}

func (r *V2WorkflowRunResult) GetDetailAsV2WorkflowRunResultGenericDetail() (*V2WorkflowRunResultGenericDetail, error) {
	if err := r.Detail.castData(); err != nil {
		return nil, err
	}
	i, ok := r.Detail.Data.(*V2WorkflowRunResultGenericDetail)
	if !ok {
		var ii V2WorkflowRunResultGenericDetail
		ii, ok = r.Detail.Data.(V2WorkflowRunResultGenericDetail)
		if ok {
			i = &ii
		}
	}
	if !ok {
		return nil, errors.New("unable to cast detail as V2WorkflowRunResultGenericDetail")
	}
	return i, nil
}

func (r *V2WorkflowRunResult) Name() string {
	switch r.Type {
	case V2WorkflowRunResultTypeGeneric:
		detail, err := r.GetDetailAsV2WorkflowRunResultGenericDetail()
		if err == nil {
			return string(r.Type) + ":" + detail.Name
		}
	case V2WorkflowRunResultTypeDocker:
		detail, err := r.GetDetailAsV2WorkflowRunResultDockerDetail()
		if err == nil {
			return string(r.Type) + ":" + detail.Name
		}
	case V2WorkflowRunResultTypeVariable:
		detail, ok := r.Detail.Data.(*V2WorkflowRunResultVariableDetail)
		if ok {
			return string(r.Type) + ":" + detail.Name
		}
	}
	return string(r.Type) + ":" + r.ID
}

func (r *V2WorkflowRunResult) Typ() string {
	if r.Detail.Type == "" {
		r.Detail.Type = reflect.TypeOf(r.Detail.Data).Name()
	}
	return string(r.Type) + ":" + r.Detail.Type
}

const (
	V2WorkflowRunResultStatusPending   = "PENDING"
	V2WorkflowRunResultStatusCompleted = "COMPLETED"
)

type V2WorkflowRunResultArtifactManagerMetadata map[string]string

func (m *V2WorkflowRunResultArtifactManagerMetadata) Set(k, v string) {
	(*m)[k] = v
}

func (m *V2WorkflowRunResultArtifactManagerMetadata) Get(k string) string {
	if m == nil {
		return ""
	}
	return (*m)[k]
}

func (x V2WorkflowRunResultArtifactManagerMetadata) Value() (driver.Value, error) {
	j, err := json.Marshal(x)
	return j, WrapError(err, "cannot marshal V2WorkflowRunResultArtifactManagerMetadata")
}

func (x *V2WorkflowRunResultArtifactManagerMetadata) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal([]byte(source), x), "cannot unmarshal V2WorkflowRunResultArtifactManagerMetadata")
}

type V2WorkflowRunResultDetail struct {
	Data interface{}
	Type string
}

func (s *V2WorkflowRunResultDetail) castData() error {
	switch s.Type {
	case "V2WorkflowRunResultGenericDetail":
		var detail = new(V2WorkflowRunResultGenericDetail)
		if err := mapstructure.Decode(s.Data, &detail); err != nil {
			return WrapError(err, "cannot unmarshal V2WorkflowRunResultGenericDetail")
		}
		s.Data = detail
		return nil
	case "V2WorkflowRunResultVariableDetail":
		var detail = new(V2WorkflowRunResultVariableDetail)
		if err := mapstructure.Decode(s.Data, &detail); err != nil {
			return WrapError(err, "cannot unmarshal V2WorkflowRunResultVariableDetail")
		}
		s.Data = detail
		return nil
	case "V2WorkflowRunResultDockerDetail":
		var detail = new(V2WorkflowRunResultDockerDetail)
		if err := mapstructure.Decode(s.Data, &detail); err != nil {
			return WrapError(err, "cannot unmarshal V2WorkflowRunResultDockerDetail")
		}
		s.Data = detail
		return nil
	case "V2WorkflowRunResultHelmDetail":
		var detail = new(V2WorkflowRunResultHelmDetail)
		if err := mapstructure.Decode(s.Data, &detail); err != nil {
			return WrapError(err, "cannot unmarshal V2WorkflowRunResultHelmDetail")
		}
		s.Data = detail
		return nil
	default:
		return errors.Errorf("unsupported type %q", s.Type)
	}
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *V2WorkflowRunResultDetail) UnmarshalJSON(source []byte) error {
	var content = struct {
		Data interface{}
		Type string
	}{}
	if err := JSONUnmarshal(source, &content); err != nil {
		return WrapError(err, "cannot unmarshal V2WorkflowRunResultDetail")
	}
	s.Data = content.Data
	s.Type = content.Type

	if err := s.castData(); err != nil {
		return err
	}

	return nil
}

// MarshalJSON implements json.Marshaler.
func (s *V2WorkflowRunResultDetail) MarshalJSON() ([]byte, error) {
	if s.Type == "" {
		s.Type = reflect.TypeOf(s.Data).Name()
	}

	var content = struct {
		Data interface{}
		Type string
	}{
		Data: s.Data,
		Type: s.Type,
	}

	btes, _ := json.Marshal(content)
	return btes, nil
}

var (
	_ json.Marshaler   = new(V2WorkflowRunResultDetail)
	_ json.Unmarshaler = new(V2WorkflowRunResultDetail)
)

// Value returns driver.Value from V2WorkflowRunResultDetail
func (s V2WorkflowRunResultDetail) Value() (driver.Value, error) {
	j, err := json.Marshal(s)
	return j, WrapError(err, "cannot marshal V2WorkflowRunResultDetail")
}

// Scan V2WorkflowRunResultDetail
func (s *V2WorkflowRunResultDetail) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	if err := JSONUnmarshal(source, s); err != nil {
		return WrapError(err, "cannot unmarshal V2WorkflowRunResultDetail")
	}
	return nil
}

type V2WorkflowRunResultType string

const (
	V2WorkflowRunResultTypeCoverage = "coverage"
	V2WorkflowRunResultTypeTest     = "tests"
	V2WorkflowRunResultTypeRelease  = "release"
	V2WorkflowRunResultTypeGeneric  = "generic"
	V2WorkflowRunResultTypeVariable = "variable"
	V2WorkflowRunResultTypeDocker   = "docker"
	V2WorkflowRunResultTypeHelm     = "helm"
	// Other values may be instantiated from Artifactory Manager repository type
)

type V2WorkflowRunResultGenericDetail struct {
	Name   string      `json:"name" mapstructure:"name"`
	Size   int64       `json:"size" mapstructure:"size"`
	Mode   os.FileMode `json:"mode" mapstructure:"mode"`
	MD5    string      `json:"md5" mapstructure:"md5"`
	SHA1   string      `json:"sha1" mapstructure:"sha1"`
	SHA256 string      `json:"sha256" mapstructure:"sha256"`
}

type V2WorkflowRunResultDockerDetail struct {
	Name         string `json:"name" mapstructure:"name"`
	ID           string `json:"id" mapstructure:"id"`
	HumanSize    string `json:"human_size" mapstructure:"human_size"`
	HumanCreated string `json:"human_created" mapstructure:"human_created"`
}

type V2WorkflowRunResultHelmDetail struct {
	Name       string `json:"name" mapstructure:"name"`
	AppVersion string `json:"appVersion" mapstructure:"appVersion"`
}

type V2WorkflowRunResultVariableDetail struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type V2WorkflowRunSearchFilter struct {
	Key     string   `json:"key"`
	Options []string `json:"options"`
	Example string   `json:"example"`
}
