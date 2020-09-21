package hatchery

import (
	"context"
	"crypto/rsa"

	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
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

// WorkerJWTClaims is the specific claims format for Worker JWT
type WorkerJWTClaims struct {
	jwt.StandardClaims
	Worker SpawnArguments
}

// SpawnArguments contains arguments to func SpawnWorker
type SpawnArguments struct {
	WorkerName   string `json:"worker_model"`
	WorkerToken  string
	Model        *sdk.Model        `json:"model"`
	JobName      string            `json:"job_name"`
	JobID        int64             `json:"job_id"`
	NodeRunID    int64             `json:"node_run_id"`
	NodeRunName  string            `json:"node_run_name"`
	Requirements []sdk.Requirement `json:"requirements"`
	RegisterOnly bool              `json:"register_only"`
	HatcheryName string            `json:"hatchery_name"`
	ProjectKey   string            `json:"project_key"`
	WorkflowName string            `json:"workflow_name"`
	WorkflowID   int64             `json:"workflow_id"`
	RunID        int64             `json:"run_id"`
}

func (s *SpawnArguments) ModelName() string {
	if s.Model != nil {
		return s.Model.Group.Name + "/" + s.Model.Name
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
	CanSpawn(ctx context.Context, model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool
	WorkersStarted(ctx context.Context) []string
	Service() *sdk.Service
	CDSClient() cdsclient.Interface
	Configuration() service.HatcheryCommonConfiguration
	Serve(ctx context.Context) error
	PanicDumpDirectory() (string, error)
	GetPrivateKey() *rsa.PrivateKey
	GetLogger() *logrus.Logger
	GetGoRoutines() *sdk.GoRoutines
}

type InterfaceWithModels interface {
	Interface
	WorkersStartedByModel(ctx context.Context, model *sdk.Model) int
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
