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
	Ref           string                 `json:"ref,omitempty"`
	Sha           string                 `json:"sha,omitempty"`
	Payload       map[string]interface{} `json:"payload"`
	HookType      string                 `json:"hook_type"`
	EntityUpdated string                 `json:"entity_updated"`
	SemverCurrent string                 `json:"semver_current"`
	SemverNext    string                 `json:"semver_next"`
	ChangeSets    []string               `json:"changesets"`
	Cron          string                 `json:"cron"`
	CronTimezone  string                 `json:"cron_timezone"`
}

type V2WorkflowRun struct {
	ID            string                 `json:"id" db:"id" cli:"id" action_metadata:"workflow-run-id"`
	ProjectKey    string                 `json:"project_key" db:"project_key" action_metadata:"project-key"`
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
	Contexts      WorkflowRunContext     `json:"contexts" db:"contexts"`
	RunEvent      V2WorkflowRunEvent     `json:"event" db:"event"`
	RunJobEvent   V2WorkflowRunJobEvents `json:"job_events" db:"job_event"`
	RetentionDate time.Time              `json:"retention_date,omitempty" db:"retention_date" cli:"-"`
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
	HookType      string                 `json:"hook_type"`
	EventName     string                 `json:"event_name"`
	Ref           string                 `json:"ref"`
	Sha           string                 `json:"sha"`
	SemverCurrent string                 `json:"semver_current"`
	SemverNext    string                 `json:"semver_next"`
	ChangeSets    []string               `json:"changesets"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
	EntityUpdated string                 `json:"entity_updated,omitempty"`
	Cron          string                 `json:"cron,omitempty"`
	CronTimezone  string                 `json:"timezone,omitempty"`
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
	Conclusion V2WorkflowRunJobStatus `json:"conclusion"` // result of a step after 'continue-on-error'
	Outcome    V2WorkflowRunJobStatus `json:"outcome"`    // result of a step before 'continue-on-error'
	Outputs    JobResultOutput        `json:"outputs"`
	Started    time.Time              `json:"started"`
	Ended      time.Time              `json:"ended"`
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
		for _, status := range stage.Jobs {
			if !status.IsTerminated() {
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
	Jobs     map[string]V2WorkflowRunJobStatus
	Ended    bool
}

func (w V2WorkflowRun) GetStages() WorkflowRunStages {
	stages := WorkflowRunStages{}
	for k, s := range w.WorkflowData.Workflow.Stages {
		stages[k] = WorkflowRunStage{WorkflowStage: s, Jobs: make(map[string]V2WorkflowRunJobStatus)}
	}
	if len(stages) == 0 {
		return stages
	}
	for jobID, job := range w.WorkflowData.Workflow.Jobs {
		stages[job.Stage].Jobs[jobID] = V2WorkflowRunJobStatusUnknown
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
}

func (r *V2WorkflowRunResult) GetDetail() (any, error) {
	if err := r.Detail.castData(); err != nil {
		return nil, err
	}
	return r.Detail.Data, nil
}

func (r *V2WorkflowRunResult) GetDetailAsV2WorkflowRunResultVariableDetail() (*V2WorkflowRunResultVariableDetail, error) {
	if err := r.Detail.castData(); err != nil {
		return nil, err
	}
	i, ok := r.Detail.Data.(*V2WorkflowRunResultVariableDetail)
	if !ok {
		var ii V2WorkflowRunResultVariableDetail
		ii, ok = r.Detail.Data.(V2WorkflowRunResultVariableDetail)
		if ok {
			i = &ii
		}
	}
	if !ok {
		return nil, errors.New("unable to cast detail as V2WorkflowRunResultArsenalDeploymentDetail")
	}
	return i, nil
}

func (r *V2WorkflowRunResult) GetDetailAsV2WorkflowRunResultArsenalDeploymentDetail() (*V2WorkflowRunResultArsenalDeploymentDetail, error) {
	if err := r.Detail.castData(); err != nil {
		return nil, err
	}
	i, ok := r.Detail.Data.(*V2WorkflowRunResultArsenalDeploymentDetail)
	if !ok {
		var ii V2WorkflowRunResultArsenalDeploymentDetail
		ii, ok = r.Detail.Data.(V2WorkflowRunResultArsenalDeploymentDetail)
		if ok {
			i = &ii
		}
	}
	if !ok {
		return nil, errors.New("unable to cast detail as V2WorkflowRunResultArsenalDeploymentDetail")
	}
	return i, nil
}

func (r *V2WorkflowRunResult) GetDetailAsV2WorkflowRunResultPythonDetail() (*V2WorkflowRunResultPythonDetail, error) {
	if err := r.Detail.castData(); err != nil {
		return nil, err
	}
	i, ok := r.Detail.Data.(*V2WorkflowRunResultPythonDetail)
	if !ok {
		var ii V2WorkflowRunResultPythonDetail
		ii, ok = r.Detail.Data.(V2WorkflowRunResultPythonDetail)
		if ok {
			i = &ii
		}
	}
	if !ok {
		return nil, errors.New("unable to cast detail as V2WorkflowRunResultPythonDetail")
	}
	return i, nil
}

func (r *V2WorkflowRunResult) GetDetailAsV2WorkflowRunResultDebianDetail() (*V2WorkflowRunResultDebianDetail, error) {
	if err := r.Detail.castData(); err != nil {
		return nil, err
	}
	i, ok := r.Detail.Data.(*V2WorkflowRunResultDebianDetail)
	if !ok {
		var ii V2WorkflowRunResultDebianDetail
		ii, ok = r.Detail.Data.(V2WorkflowRunResultDebianDetail)
		if ok {
			i = &ii
		}
	}
	if !ok {
		return nil, errors.New("unable to cast detail as V2WorkflowRunResultDebianDetail")
	}
	return i, nil
}

func (r *V2WorkflowRunResult) GetDetailAsV2WorkflowRunResultHelmDetail() (*V2WorkflowRunResultHelmDetail, error) {
	if err := r.Detail.castData(); err != nil {
		return nil, err
	}
	i, ok := r.Detail.Data.(*V2WorkflowRunResultHelmDetail)
	if !ok {
		var ii V2WorkflowRunResultHelmDetail
		ii, ok = r.Detail.Data.(V2WorkflowRunResultHelmDetail)
		if ok {
			i = &ii
		}
	}
	if !ok {
		return nil, errors.New("unable to cast detail as V2WorkflowRunResultHelmDetail")
	}
	return i, nil
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

func (r *V2WorkflowRunResult) GetDetailAsV2WorkflowRunResultTestDetail() (*V2WorkflowRunResultTestDetail, error) {
	if err := r.Detail.castData(); err != nil {
		return nil, err
	}
	i, ok := r.Detail.Data.(*V2WorkflowRunResultTestDetail)
	if !ok {
		var ii V2WorkflowRunResultTestDetail
		ii, ok = r.Detail.Data.(V2WorkflowRunResultTestDetail)
		if ok {
			i = &ii
		}
	}
	if !ok {
		return nil, errors.New("unable to cast detail as V2WorkflowRunResultTestDetail")
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

func (r *V2WorkflowRunResult) GetDetailAsV2WorkflowRunResultReleaseDetail() (*V2WorkflowRunResultReleaseDetail, error) {
	if err := r.Detail.castData(); err != nil {
		return nil, err
	}
	i, ok := r.Detail.Data.(*V2WorkflowRunResultReleaseDetail)
	if !ok {
		var ii V2WorkflowRunResultReleaseDetail
		ii, ok = r.Detail.Data.(V2WorkflowRunResultReleaseDetail)
		if ok {
			i = &ii
		}
	}
	if !ok {
		return nil, errors.New("unable to cast detail as V2WorkflowRunResultReleaseDetail")
	}
	return i, nil
}

func (r *V2WorkflowRunResult) Name() string {
	switch r.Type {
	case V2WorkflowRunResultTypeTest:
		detail, err := r.GetDetailAsV2WorkflowRunResultTestDetail()
		if err == nil {
			return string(r.Type) + ":" + detail.Name
		}
	case V2WorkflowRunResultTypeGeneric, V2WorkflowRunResultTypeCoverage:
		detail, err := r.GetDetailAsV2WorkflowRunResultGenericDetail()
		if err == nil {
			return string(r.Type) + ":" + detail.Name
		}
	case V2WorkflowRunResultTypePython:
		detail, err := r.GetDetailAsV2WorkflowRunResultPythonDetail()
		if err == nil {
			return string(r.Type) + ":" + detail.Name + ":" + detail.Version
		}
	case V2WorkflowRunResultTypeDebian:
		detail, err := r.GetDetailAsV2WorkflowRunResultDebianDetail()
		if err == nil {
			return string(r.Type) + ":" + detail.Name
		}
	case V2WorkflowRunResultTypeDocker:
		detail, err := r.GetDetailAsV2WorkflowRunResultDockerDetail()
		if err == nil {
			return string(r.Type) + ":" + detail.Name
		}
	case V2WorkflowRunResultTypeVariable:
		detail, err := r.GetDetailAsV2WorkflowRunResultVariableDetail()
		if err == nil {
			return string(r.Type) + ":" + detail.Name
		}
	case V2WorkflowRunResultTypeArsenalDeployment:
		detail, err := r.GetDetailAsV2WorkflowRunResultArsenalDeploymentDetail()
		if err == nil {
			return string(r.Type) + ":" + detail.DeploymentName
		}
	case V2WorkflowRunResultTypeHelm:
		detail, err := r.GetDetailAsV2WorkflowRunResultHelmDetail()
		if err == nil {
			return string(r.Type) + ":" + detail.Name + ":" + detail.ChartVersion
		}
	case V2WorkflowRunResultTypeRelease:
		detail, err := r.GetDetailAsV2WorkflowRunResultReleaseDetail()
		if err == nil {
			return string(r.Type) + ":" + detail.Name + ":" + detail.Version
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

type V2WorkflowRunResultDetail struct {
	Data interface{} `json:"data"`
	Type string      `json:"type"`
}

func (s *V2WorkflowRunResultDetail) castData() error {
	switch s.Type {
	case "V2WorkflowRunResultTestDetail":
		var detail = new(V2WorkflowRunResultTestDetail)
		if err := mapstructure.Decode(s.Data, &detail); err != nil {
			return WrapError(err, "cannot unmarshal V2WorkflowRunResultTestDetail")
		}
		s.Data = detail
		return nil
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
	case "V2WorkflowRunResultDebianDetail":
		var detail = new(V2WorkflowRunResultDebianDetail)
		if err := mapstructure.Decode(s.Data, &detail); err != nil {
			return WrapError(err, "cannot unmarshal V2WorkflowRunResultDebianDetail")
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
	case "V2WorkflowRunResultPythonDetail":
		var detail = new(V2WorkflowRunResultPythonDetail)
		if err := mapstructure.Decode(s.Data, &detail); err != nil {
			return WrapError(err, "cannot unmarshal V2WorkflowRunResultPythonDetail")
		}
		s.Data = detail
		return nil
	case "V2WorkflowRunResultArsenalDeploymentDetail":
		var detail = new(V2WorkflowRunResultArsenalDeploymentDetail)
		if err := mapstructure.Decode(s.Data, &detail); err != nil {
			return WrapError(err, "cannot unmarshal V2WorkflowRunResultArsenalDeploymentDetail")
		}
		s.Data = detail
		return nil
	case "V2WorkflowRunResultReleaseDetail":
		var detail = new(V2WorkflowRunResultReleaseDetail)
		decoderConfig := &mapstructure.DecoderConfig{
			Metadata: nil,
			Result:   &detail,
		}
		// Here is the trick to transform the map to a json.RawMessage for the SBOM itself
		decoderConfig.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
				if f.Kind() != reflect.Map {
					return data, nil
				}
				result := reflect.New(t).Interface()
				_, ok := result.(*json.RawMessage)
				if !ok {
					return data, nil
				}
				btes, err := json.Marshal(data)
				if err != nil {
					return nil, err
				}
				return json.RawMessage(btes), nil
			},
		)
		decoder, err := mapstructure.NewDecoder(decoderConfig)
		if err != nil {
			panic(err)
		}
		if err := decoder.Decode(s.Data); err != nil {
			return WrapError(err, "cannot unmarshal V2WorkflowRunResultReleaseDetail")
		}
		s.Data = *detail
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
		Data interface{} `json:"data"`
		Type string      `json:"type"`
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
	V2WorkflowRunResultTypeCoverage          = "coverage"
	V2WorkflowRunResultTypeTest              = "tests"
	V2WorkflowRunResultTypeRelease           = "release"
	V2WorkflowRunResultTypeGeneric           = "generic"
	V2WorkflowRunResultTypeVariable          = "variable"
	V2WorkflowRunResultTypeDocker            = "docker"
	V2WorkflowRunResultTypeDebian            = "debian"
	V2WorkflowRunResultTypePython            = "python"
	V2WorkflowRunResultTypeArsenalDeployment = "deployment"
	V2WorkflowRunResultTypeHelm              = "helm"
	// Other values may be instantiated from Artifactory Manager repository type
)

type V2WorkflowRunResultTestDetail struct {
	Name        string           `json:"name" mapstructure:"name"`
	Size        int64            `json:"size" mapstructure:"size"`
	Mode        os.FileMode      `json:"mode" mapstructure:"mode"`
	MD5         string           `json:"md5" mapstructure:"md5"`
	SHA1        string           `json:"sha1" mapstructure:"sha1"`
	SHA256      string           `json:"sha256" mapstructure:"sha256"`
	TestsSuites JUnitTestsSuites `json:"tests_suites" mapstructure:"tests_suites"`
	TestStats   TestsStats       `json:"tests_stats" mapstructure:"tests_stats"`
}

type V2WorkflowRunResultGenericDetail struct {
	Name   string      `json:"name" mapstructure:"name"`
	Size   int64       `json:"size" mapstructure:"size"`
	Mode   os.FileMode `json:"mode" mapstructure:"mode"`
	MD5    string      `json:"md5" mapstructure:"md5"`
	SHA1   string      `json:"sha1" mapstructure:"sha1"`
	SHA256 string      `json:"sha256" mapstructure:"sha256"`
}

type V2WorkflowRunResultArsenalDeploymentDetail struct {
	IntegrationName string                              `json:"integration_name" mapstructure:"integration_name"`
	DeploymentID    string                              `json:"deployment_id" mapstructure:"deployment_id"`
	DeploymentName  string                              `json:"deployment_name" mapstructure:"deployment_name"`
	StackID         string                              `json:"stack_id" mapstructure:"stack_id"`
	StackName       string                              `json:"stack_name" mapstructure:"stack_name"`
	StackPlatform   string                              `json:"stack_platform" mapstructure:"stack_platform"`
	Namespace       string                              `json:"namespace" mapstructure:"namespace"`
	Version         string                              `json:"version" mapstructure:"version"`
	Alternative     *ArsenalDeploymentDetailAlternative `json:"alternative" mapstructure:"alternative"`
}

type ArsenalDeploymentDetailAlternative struct {
	Name    string                 `json:"name" mapstructure:"name"`
	From    string                 `json:"from,omitempty" mapstructure:"from"`
	Config  map[string]interface{} `json:"config" mapstructure:"config"`
	Options map[string]interface{} `json:"options,omitempty" mapstructure:"options"`
}

type V2WorkflowRunResultDockerDetail struct {
	Name         string `json:"name" mapstructure:"name"`
	ID           string `json:"id" mapstructure:"id"`
	HumanSize    string `json:"human_size" mapstructure:"human_size"`
	HumanCreated string `json:"human_created" mapstructure:"human_created"`
}

type V2WorkflowRunResultDebianDetail struct {
	Name          string   `json:"name" mapstructure:"name"`
	Size          int64    `json:"size" mapstructure:"size"`
	MD5           string   `json:"md5" mapstructure:"md5"`
	SHA1          string   `json:"sha1" mapstructure:"sha1"`
	SHA256        string   `json:"sha256" mapstructure:"sha256"`
	Components    []string `json:"components" mapstructure:"components"`
	Distributions []string `json:"distributions" mapstructure:"distributions"`
	Architectures []string `json:"architectures" mapstructure:"architectures"`
}

type V2WorkflowRunResultPythonDetail struct {
	Name      string `json:"name" mapstructure:"name"`
	Version   string `json:"version" mapstructure:"version"`
	Extension string `json:"extension" mapstructure:"extension"`
}

type V2WorkflowRunResultHelmDetail struct {
	Name         string `json:"name" mapstructure:"name"`
	AppVersion   string `json:"appVersion" mapstructure:"appVersion"`
	ChartVersion string `json:"chartVersion" mapstructure:"chartVersion"`
}

type V2WorkflowRunResultVariableDetail struct {
	Name  string `json:"name" mapstructure:"name"`
	Value string `json:"value" mapstructure:"value"`
}

type V2WorkflowRunResultReleaseDetail struct {
	Name    string          `json:"name" mapstructure:"name"`
	Version string          `json:"version" mapstructure:"version"`
	SBOM    json.RawMessage `json:"sbom" mapstructure:"sbom"`
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
}
