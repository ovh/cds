package hatchery

import (
	"context"
	"crypto/rsa"

	jwt "github.com/golang-jwt/jwt"
	"github.com/rockbears/log"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

const (
	LabelServiceJobID        = "CDS_JOB_ID"
	LabelServiceProjectKey   = "CDS_PROJECT_KEY"
	LabelServiceWorkflowName = "CDS_WORKFLOW_NAME"
	LabelServiceWorkflowID   = "CDS_WORKFLOW_ID"
	LabelServiceRunID        = "CDS_WORKFLOW_RUN_ID"
	LabelServiceNodeRunName  = "CDS_NODE_RUN_NAME"
	LabelServiceNodeRunID    = "CDS_NODE_RUN_ID"
	LabelServiceJobName      = "CDS_JOB_NAME"
	LabelServiceID           = "CDS_SERVICE_ID"
	LabelServiceReqName      = "CDS_SERVICE_NAME"
)

var (
	LogFieldJobID        = log.Field("action_metadata_job_id")
	LogFieldStep         = log.Field("hatchery_step")
	LogFieldStepDelay    = log.Field("hatchery_step_delay_num")
	LogFieldProjectID    = log.Field("worker_project_id")
	LogFieldProject      = log.Field("worker_project")
	LogFieldWorkflow     = log.Field("worker_workflow")
	LogFieldNodeRunID    = log.Field("worker_node_run_id")
	LogFieldNodeRun      = log.Field("worker_node_run")
	LogFieldModel        = log.Field("worker_model")
	LogFieldServiceCount = log.Field("worker_service_count_num")
)

func init() {
	log.RegisterField(LogFieldJobID)
	log.RegisterField(LogFieldStep)
	log.RegisterField(LogFieldStepDelay)
	log.RegisterField(LogFieldProjectID)
	log.RegisterField(LogFieldProject)
	log.RegisterField(LogFieldWorkflow)
	log.RegisterField(LogFieldNodeRunID)
	log.RegisterField(LogFieldNodeRun)
	log.RegisterField(LogFieldModel)
	log.RegisterField(LogFieldServiceCount)
}

// WorkerJWTClaims is the specific claims format for Worker JWT
type WorkerJWTClaims struct {
	jwt.StandardClaims
	Worker SpawnArgumentsJWT
}

type SpawnArgumentsJWT struct {
	WorkerName string `json:"worker_model,omitempty"`
	Model      struct {
		ID   int64  `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"model,omitempty"`
	JobID        int64  `json:"job_id,omitempty"`
	RegisterOnly bool   `json:"register_only"`
	HatcheryName string `json:"hatchery_name,omitempty"`
}

func (s SpawnArgumentsJWT) Validate() error {
	if s.WorkerName == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unauthorized to register a worker without a name")
	}
	if !s.RegisterOnly && s.JobID == 0 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unauthorized to register a worker for a job without a JobID")
	}
	if s.RegisterOnly && s.JobID > 0 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unauthorized to register only worker with a JobID")
	}
	return nil
}

// WorkerJWTClaims is the specific claims format for Worker JWT
type WorkerJWTClaimsV2 struct {
	jwt.StandardClaims
	Worker SpawnArgumentsJWTV2
}

type SpawnArgumentsJWTV2 struct {
	WorkerName   string `json:"worker_model,omitempty"`
	ModelName    string `json:"model_name,omitempty"`
	RunJobID     string `json:"run_job_id,omitempty"`
	HatcheryName string `json:"hatchery_name,omitempty"`
}

func (s SpawnArgumentsJWTV2) Validate() error {
	if s.WorkerName == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unauthorized to register a worker without a name")
	}
	if !sdk.IsValidUUID(s.RunJobID) {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unauthorized to register a worker for a run_job without a valid ID")
	}
	return nil
}

// SpawnArguments contains arguments to func SpawnWorker
type SpawnArguments struct {
	WorkerName   string `json:"worker_model"`
	WorkerToken  string
	Model        sdk.WorkerStarterWorkerModel `json:"model"`
	JobName      string                       `json:"job_name"`
	JobID        string                       `json:"job_id"`
	NodeRunID    int64                        `json:"node_run_id"`
	NodeRunName  string                       `json:"node_run_name"`
	Requirements []sdk.Requirement            `json:"requirements"`
	RegisterOnly bool                         `json:"register_only"`
	HatcheryName string                       `json:"hatchery_name"`
	ProjectKey   string                       `json:"project_key"`
	WorkflowName string                       `json:"workflow_name"`
	WorkflowID   int64                        `json:"workflow_id"`
	RunID        string                       `json:"run_id"`
}

func (s *SpawnArguments) ModelName() string {
	if s.Model.ModelV1 != nil {
		return s.Model.GetFullPath()
	} else if s.Model.ModelV2 != nil {
		return s.Model.GetName()
	}
	return ""
}

// Interface describe an interface for each hatchery mode
// Init create new clients for different api
// SpawnWorker creates a new vm instance
// CanSpawn return wether or not hatchery can spawn model
// WorkersStartedByModel returns the number of instances of given model started but not necessarily register on CDS yet
// WorkersStarted returns the number of instances started but not necessarily register on CDS yet
// Hatchery returns hatchery instance
// Client returns cdsclient instance
// ModelType returns type of hatchery
// NeedRegistration return true if worker model need regsitration
// ID returns hatchery id
type Interface interface {
	Name() string
	Type() string
	InitHatchery(ctx context.Context) error
	SpawnWorker(ctx context.Context, spawnArgs SpawnArguments) error
	CanSpawn(ctx context.Context, model sdk.WorkerStarterWorkerModel, jobID string, requirements []sdk.Requirement) bool
	WorkersStarted(ctx context.Context) ([]string, error)
	Service() *sdk.Service
	CDSClient() cdsclient.Interface
	CDSClientV2() cdsclient.HatcheryServiceClient
	Configuration() service.HatcheryCommonConfiguration
	Serve(ctx context.Context) error
	GetPrivateKey() *rsa.PrivateKey
	GetGoRoutines() *sdk.GoRoutines
}

type InterfaceWithModels interface {
	Interface
	ModelType() string
	NeedRegistration(ctx context.Context, model *sdk.Model) bool
	WorkerModelsEnabled() ([]sdk.Model, error)
	WorkerModelSecretList(sdk.Model) (sdk.WorkerModelSecrets, error)
}

type Metrics struct {
	Jobs               *stats.Int64Measure
	JobsWebsocket      *stats.Int64Measure
	SpawnedWorkers     *stats.Int64Measure
	PendingWorkers     *stats.Int64Measure
	RegisteringWorkers *stats.Int64Measure
	CheckingWorkers    *stats.Int64Measure
	WaitingWorkers     *stats.Int64Measure
	BuildingWorkers    *stats.Int64Measure
	DisabledWorkers    *stats.Int64Measure
}

type JobIdentifiers struct {
	ServiceID  int64
	JobID      int64
	NodeRunID  int64
	RunID      int64
	WorkflowID int64
}
