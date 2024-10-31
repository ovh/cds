package sdk

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rockbears/log"
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
	EventName     WorkflowHookEventName  `json:"event_name"`
	Ref           string                 `json:"ref,omitempty"`
	Sha           string                 `json:"sha,omitempty"`
	PullrequestID int64                  `json:"pr_id,omitempty"`
	CommitMessage string                 `json:"commit_message,omitempty"`
	Payload       map[string]interface{} `json:"payload"`
	HookType      string                 `json:"hook_type"`
	EntityUpdated string                 `json:"entity_updated"`
	SemverCurrent string                 `json:"semver_current"`
	SemverNext    string                 `json:"semver_next"`
	ChangeSets    []string               `json:"changesets"`
	Cron          string                 `json:"cron"`
	CronTimezone  string                 `json:"cron_timezone"`
	AdminMFA      bool                   `json:"admin_mfa"`
	WorkflowRun   string                 `json:"workflow_run"`
	WorkflowRunID string                 `json:"workflow_run_id"`
}

type V2WorkflowRun struct {
	ID            string                 `json:"id" db:"id" cli:"id" action_metadata:"workflow-run-id"`
	ProjectKey    string                 `json:"project_key" cli:"project_key" db:"project_key" action_metadata:"project-key"`
	VCSServerID   string                 `json:"vcs_server_id" db:"vcs_server_id"`
	VCSServer     string                 `json:"vcs_server" db:"vcs_server" action_metadata:"vcs-server"`
	RepositoryID  string                 `json:"repository_id" db:"repository_id"`
	Repository    string                 `json:"repository" db:"repository" action_metadata:"repository-identifier"`
	WorkflowName  string                 `json:"workflow_name" db:"workflow_name" cli:"workflow_name" action_metadata:"workflow-name"`
	WorkflowSha   string                 `json:"workflow_sha" db:"workflow_sha"`
	WorkflowRef   string                 `json:"workflow_ref" db:"workflow_ref"`
	Status        V2WorkflowRunStatus    `json:"status" db:"status" cli:"status"`
	RunNumber     int64                  `json:"run_number" db:"run_number" cli:"run_number" action_metadata:"run-number"`
	RunAttempt    int64                  `json:"run_attempt" db:"run_attempt"`
	Started       time.Time              `json:"started" db:"started" cli:"started"`
	LastModified  time.Time              `json:"last_modified" db:"last_modified" cli:"last_modified"`
	ToDelete      bool                   `json:"to_delete" db:"to_delete"`
	WorkflowData  V2WorkflowRunData      `json:"workflow_data" db:"workflow_data"`
	UserID        string                 `json:"user_id" db:"user_id"`
	Username      string                 `json:"username" db:"username" cli:"username" action_metadata:"username"`
	AdminMFA      bool                   `json:"admin_mfa" db:"admin_mfa" cli:"admin_mfa"`
	Contexts      WorkflowRunContext     `json:"contexts" db:"contexts"`
	RunEvent      V2WorkflowRunEvent     `json:"event" db:"event"`
	RunJobEvent   V2WorkflowRunJobEvents `json:"job_events" db:"job_event"`
	RetentionDate time.Time              `json:"retention_date,omitempty" db:"retention_date" cli:"-"`
	Annotations   WorkflowRunAnnotations `json:"annotations,omitempty" db:"annotations" cli:"-"`
}

type V2WorkflowRunStatus string

const (
	V2WorkflowRunStatusSkipped  V2WorkflowRunStatus = "Skipped"
	V2WorkflowRunStatusFail     V2WorkflowRunStatus = "Fail"
	V2WorkflowRunStatusSuccess  V2WorkflowRunStatus = "Success"
	V2WorkflowRunStatusStopped  V2WorkflowRunStatus = "Stopped"
	V2WorkflowRunStatusBuilding V2WorkflowRunStatus = "Building"
	V2WorkflowRunStatusCrafting V2WorkflowRunStatus = "Crafting"
)

func (s V2WorkflowRunStatus) IsTerminated() bool {
	switch s {
	case V2WorkflowRunStatusBuilding, V2WorkflowRunStatusCrafting:
		return false
	}
	return true
}

type WorkflowRunAnnotations map[string]string

func (m WorkflowRunAnnotations) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	j, err := json.Marshal(m)
	return j, WrapError(err, "cannot marshal WorkflowRunAnnotations")
}

func (m *WorkflowRunAnnotations) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(json.Unmarshal(source, m), "cannot unmarshal WorkflowRunAnnotations")
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
	Inputs       map[string]interface{}   `json:"inputs,omitempty"`
	Jobs         JobsResultContext        `json:"jobs"`
	Needs        NeedsContext             `json:"needs"`
	Steps        StepsContext             `json:"steps"`
	Matrix       map[string]string        `json:"matrix"`
	Integrations *JobIntegrationsContexts `json:"integrations,omitempty"`
	Gate         map[string]interface{}   `json:"gate"`
	Vars         map[string]interface{}   `json:"vars"`
}

type ComputeAnnotationsContext struct {
	WorkflowRunContext
	Jobs map[string]ComputeAnnotationsJobContext `json:"jobs"`
}

type ComputeAnnotationsJobContext struct {
	Results JobResultContext `json:"results"`
	Gate    GateInputs       `json:"gate"`
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
	HookType      string                 `json:"hook_type"`
	EventName     WorkflowHookEventName  `json:"event_name"`
	Ref           string                 `json:"ref"`
	Sha           string                 `json:"sha"`
	PullRequestID int64                  `json:"pullrequest_id"`
	CommitMessage string                 `json:"commit_message"`
	SemverCurrent string                 `json:"semver_current"`
	SemverNext    string                 `json:"semver_next"`
	ChangeSets    []string               `json:"changesets"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
	EntityUpdated string                 `json:"entity_updated,omitempty"`
	Cron          string                 `json:"cron,omitempty"`
	CronTimezone  string                 `json:"timezone,omitempty"`
	WorkflowRun   string                 `json:"workflow_run"`
	WorkflowRunID string                 `json:"workflow_run_id"`
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

type V2WorkflowRunJob struct {
	ID            string                 `json:"id" db:"id"`
	JobID         string                 `json:"job_id" db:"job_id" cli:"job_id"`
	WorkflowRunID string                 `json:"workflow_run_id" db:"workflow_run_id" action_metadata:"workflow-run-id"`
	ProjectKey    string                 `json:"project_key" db:"project_key" action_metadata:"project-key"`
	WorkflowName  string                 `json:"workflow_name" db:"workflow_name" action_metadata:"workflow-name"`
	RunNumber     int64                  `json:"run_number" db:"run_number" action_metadata:"run-number"`
	RunAttempt    int64                  `json:"run_attempt" db:"run_attempt"`
	Status        V2WorkflowRunJobStatus `json:"status" db:"status" cli:"status"`
	Queued        time.Time              `json:"queued" db:"queued"`
	Scheduled     *time.Time             `json:"scheduled,omitempty" db:"scheduled"`
	Started       *time.Time             `json:"started,omitempty" db:"started"`
	Ended         *time.Time             `json:"ended,omitempty" db:"ended"`
	Job           V2Job                  `json:"job" db:"job"`
	WorkerID      string                 `json:"worker_id,omitempty" db:"worker_id"`
	WorkerName    string                 `json:"worker_name" db:"worker_name" action_metadata:"worker-name"`
	HatcheryName  string                 `json:"hatchery_name" db:"hatchery_name" action_metadata:"hatchery-name"`
	StepsStatus   JobStepsStatus         `json:"steps_status" db:"steps_status"`
	UserID        string                 `json:"user_id" db:"user_id"`
	Username      string                 `json:"username" db:"username" action_metadata:"username"`
	AdminMFA      bool                   `json:"admin_mfa" db:"admin_mfa"`
	Region        string                 `json:"region,omitempty" db:"region"`
	ModelType     string                 `json:"model_type,omitempty" db:"model_type"`
	Matrix        JobMatrix              `json:"matrix,omitempty" db:"matrix"`
	GateInputs    GateInputs             `json:"gate_inputs,omitempty" db:"gate_inputs"`
}

type V2WorkflowRunJobStatus string

const (
	V2WorkflowRunJobStatusUnknown    V2WorkflowRunJobStatus = ""
	V2WorkflowRunJobStatusWaiting    V2WorkflowRunJobStatus = "Waiting"
	V2WorkflowRunJobStatusBuilding   V2WorkflowRunJobStatus = "Building"
	V2WorkflowRunJobStatusFail       V2WorkflowRunJobStatus = "Fail"
	V2WorkflowRunJobStatusStopped    V2WorkflowRunJobStatus = "Stopped"
	V2WorkflowRunJobStatusSuccess    V2WorkflowRunJobStatus = "Success"
	V2WorkflowRunJobStatusScheduling V2WorkflowRunJobStatus = "Scheduling"
	V2WorkflowRunJobStatusSkipped    V2WorkflowRunJobStatus = "Skipped"
)

func NewV2WorkflowRunJobStatusFromString(s string) (V2WorkflowRunJobStatus, error) {
	switch s {
	case StatusFail:
		return V2WorkflowRunJobStatusFail, nil
	case StatusSuccess:
		return V2WorkflowRunJobStatusSuccess, nil
	case StatusWaiting:
		return V2WorkflowRunJobStatusWaiting, nil
	case StatusSkipped:
		return V2WorkflowRunJobStatusSkipped, nil
	case StatusScheduling:
		return V2WorkflowRunJobStatusScheduling, nil
	case StatusStopped:
		return V2WorkflowRunJobStatusStopped, nil
	case StatusBuilding:
		return V2WorkflowRunJobStatusBuilding, nil
	}
	return V2WorkflowRunJobStatusUnknown, errors.Errorf("cannot convert given status value %q to workflow run job v2 status", s)
}

func (s V2WorkflowRunJobStatus) IsTerminated() bool {
	switch s {
	case V2WorkflowRunJobStatusUnknown, V2WorkflowRunJobStatusBuilding, V2WorkflowRunJobStatusWaiting, V2WorkflowRunJobStatusScheduling:
		return false
	}
	return true
}

type JobIntegrationsContexts struct {
	ArtifactManager JobIntegrationsContext `json:"artifact_manager,omitempty"`
	Deployment      JobIntegrationsContext `json:"deployment,omitempty"`
}

func (jics *JobIntegrationsContexts) All() []JobIntegrationsContext {
	integs := make([]JobIntegrationsContext, 0)
	if jics.ArtifactManager.Name != "" {
		integs = append(integs, jics.ArtifactManager)
	}
	if jics.Deployment.Name != "" {
		integs = append(integs, jics.Deployment)
	}
	return integs
}

type JobIntegrationsContext struct {
	Name      string                      `json:"name,omitempty"`
	Config    JobIntegratiosContextConfig `json:"config,omitempty"`
	ModelName string                      `json:"model_name,omitempty"`
}

type JobIntegratiosContextConfig map[string]interface{}

func (j JobIntegrationsContext) Get(key string) string {
	keySplit := strings.Split(key, ".")
	if len(keySplit) == 1 {
		return fmt.Sprintf("%s", j.Config[key])
	}

	if j.ModelName == ArtifactoryIntegrationModelName && key == ArtifactoryConfigTokenName {
		keySplit = []string{"token_name"}
	}

	currentValue := j.Config
	for _, k := range keySplit {
		if itemMap, ok := currentValue[k].(map[string]interface{}); ok {
			currentValue = itemMap
		} else {
			return fmt.Sprintf("%s", currentValue[k])
		}
	}
	return ""
}

type JobStepsStatus map[string]JobStepStatus
type JobStepStatus struct {
	Conclusion V2WorkflowRunJobStatus `json:"conclusion"` // result of a step after 'continue-on-error'
	Outcome    V2WorkflowRunJobStatus `json:"outcome"`    // result of a step before 'continue-on-error'
	Outputs    JobResultOutput        `json:"outputs"`
	Started    time.Time              `json:"started"`
	Ended      time.Time              `json:"ended"`

	// Path Outputs
	PathOutputs StringSlice `json:"-"`
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
		if v.Conclusion == V2WorkflowRunJobStatusUnknown {
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
	RunID          string                   `json:"run_id"`
	UserID         string                   `json:"user_id"`
	IsAdminWithMFA bool                     `json:"is_admin_mfa"`
	Gate           V2WorkflowRunEnqueueGate `json:"gate"`
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
	Status V2WorkflowRunJobStatus `json:"status"`
	Error  string                 `json:"error,omitempty"`
	Time   time.Time              `json:"time"`
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
		for _, j := range stage.Jobs {
			if !j.Status.IsTerminated() {
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
	Jobs     map[string]WorkflowRunStageJob
	Ended    bool
}

type WorkflowRunStageJob struct {
	Status  V2WorkflowRunJobStatus
	IsFinal bool
}

func (w V2WorkflowRun) GetStages() WorkflowRunStages {
	stages := WorkflowRunStages{}
	for k, s := range w.WorkflowData.Workflow.Stages {
		stages[k] = WorkflowRunStage{WorkflowStage: s, Jobs: make(map[string]WorkflowRunStageJob)}
	}
	if len(stages) == 0 {
		return stages
	}
	for jobID, job := range w.WorkflowData.Workflow.Jobs {
		isFinalJob := true
		for _, existingJob := range w.WorkflowData.Workflow.Jobs {
			if job.Stage == existingJob.Stage && IsInArray(jobID, existingJob.Needs) {
				isFinalJob = false
				break
			}
		}
		stages[job.Stage].Jobs[jobID] = WorkflowRunStageJob{IsFinal: isFinalJob, Status: V2WorkflowRunJobStatusUnknown}
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
	DataSync                       *WorkflowRunResultSync                      `json:"sync" db:"sync"`
	Status                         string                                      `json:"status" db:"status"`
	// Below are computed fields and displayed on the UI
	Label      string                                       `json:"label,omitempty" db:"-"`
	Identifier string                                       `json:"identifier,omitempty" db:"-"`
	Metadata   map[string]V2WorkflowRunResultDetailMetadata `json:"metadata,omitempty" db:"-"`
}

func (r *V2WorkflowRunResult) ComputedFields() {
	if err := r.CastDetail(); err != nil {
		log.ErrorWithStackTrace(context.Background(), err)
		return
	}
	r.Identifier = r.Name()
	r.Label = r.GetLabel()
	r.Metadata = r.GetMetadata()
}

func (r *V2WorkflowRunResult) GetLabel() string {
	detail, err := r.GetDetail()
	if err != nil {
		log.ErrorWithStackTrace(context.Background(), err)
		return "-"
	}
	return detail.GetLabel()
}

func (r *V2WorkflowRunResult) GetMetadata() map[string]V2WorkflowRunResultDetailMetadata {
	detail, err := r.GetDetail()
	if err != nil {
		log.ErrorWithStackTrace(context.Background(), err)
		return nil
	}
	return detail.GetMetadata()
}

func (r *V2WorkflowRunResult) GetDetail() (V2WorkflowRunResultDetailInterface, error) {
	if err := r.CastDetail(); err != nil {
		return nil, err
	}
	if x, ok := r.Detail.Data.(V2WorkflowRunResultDetailInterface); ok {
		return x, nil
	} else { // Manage interface conversion of s.Detail.Data
		value := reflect.ValueOf(r.Detail.Data)
		t := reflect.New(value.Type())
		t.Elem().Set(value)
		return t.Interface().(V2WorkflowRunResultDetailInterface), nil
	}
}

func (r *V2WorkflowRunResult) Name() string {
	detailData, err := r.GetDetail()
	if err != nil {
		log.ErrorWithStackTrace(context.Background(), err)
	}

	return string(r.Type) + ":" + detailData.GetName()
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
	V2WorkflowRunResultStatusPromoted  = "PROMOTED"
	V2WorkflowRunResultStatusReleased  = "RELEASED"
	V2WorkflowRunResultStatusCanceled  = "CANCELED"
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

type V2WorkflowRunSearchFilter struct {
	Key     string   `json:"key"`
	Options []string `json:"options"`
	Example string   `json:"example"`
}

type V2QueueJobInfo struct {
	RunJob V2WorkflowRunJob `json:"runjob"`
	Model  V2WorkerModel    `json:"model"`
}

type HookManualWorkflowRun struct {
	UserRequest    V2WorkflowRunManualRequest
	Project        string
	VCSServer      string
	Repository     string
	WorkflowRef    string
	WorkflowCommit string
	Workflow       string
	UserID         string
	Username       string
	AdminMFA       bool
}

type HookWorkflowRunEvent struct {
	Request HookWorkflowRunEventRequest

	// Data needed to process hook
	WorkflowProject    string
	WorkflowVCSServer  string
	WorkflowRepository string
	WorkflowName       string
	WorkflowRunID      string
	WorkflowStatus     V2WorkflowRunStatus
	WorkflowRef        string
}

// Request that will be sent to sub-workflow
type HookWorkflowRunEventRequest struct {
	WorkflowRun HookWorkflowRunEventRequestWorkflowRun `json:"workflow_run"`
}

type HookWorkflowRunEventRequestWorkflowRun struct {
	CDS        CDSContext                         `json:"cds"`
	Git        GitContext                         `json:"git"`
	UserID     string                             `json:"user_id"`
	UserName   string                             `json:"username"`
	Conclusion string                             `json:"conclusion"`
	CreatedAt  time.Time                          `json:"created_at"`
	Jobs       map[string]HookWorkflowRunEventJob `json:"jobs"`
}

type HookWorkflowRunEventJob struct {
	Conclusion string `json:"conclusion"`
}
