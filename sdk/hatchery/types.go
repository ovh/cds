package hatchery

import (
	"context"
	"crypto/rsa"

	"go.opencensus.io/stats"

	"github.com/dgrijalva/jwt-go"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// CommonConfiguration is the base configuration for all hatcheries
type CommonConfiguration struct {
	Name          string `toml:"name" default:"" comment:"Name of Hatchery" json:"name"`
	RSAPrivateKey string `toml:"rsaPrivateKey" default:"" comment:"The RSA Private Key used by the hatchery.\nThis is mandatory." json:"-"`
	HTTP          struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1" json:"addr"`
		Port int    `toml:"port" default:"8086" json:"port"`
	} `toml:"http" comment:"######################\n CDS Hatchery HTTP Configuration \n######################" json:"http"`
	URL string `toml:"url" default:"http://localhost:8086" comment:"URL of this Hatchery" json:"url"`
	API struct {
		HTTP struct {
			URL      string `toml:"url" default:"http://localhost:8081" commented:"true" comment:"CDS API URL" json:"url"`
			Insecure bool   `toml:"insecure" default:"false" commented:"true" comment:"sslInsecureSkipVerify, set to true if you use a self-signed SSL on CDS API" json:"insecure"`
		} `toml:"http" json:"http"`
		Token                string `toml:"token" default:"" comment:"CDS Token to reach CDS API. See https://ovh.github.io/cds/docs/components/cdsctl/token/ " json:"-"`
		RequestTimeout       int    `toml:"requestTimeout" default:"10" comment:"Request CDS API: timeout in seconds" json:"requestTimeout"`
		MaxHeartbeatFailures int    `toml:"maxHeartbeatFailures" default:"10" comment:"Maximum allowed consecutives failures on heatbeat routine" json:"maxHeartbeatFailures"`
	} `toml:"api" json:"api"`
	Provision struct {
		Disabled                  bool `toml:"disabled" default:"false" comment:"Disabled provisioning. Format:true or false" json:"disabled"`
		Frequency                 int  `toml:"frequency" default:"30" comment:"Check provisioning each n Seconds" json:"frequency"`
		MaxWorker                 int  `toml:"maxWorker" default:"10" comment:"Maximum allowed simultaneous workers" json:"maxWorker"`
		MaxConcurrentProvisioning int  `toml:"maxConcurrentProvisioning" default:"10" comment:"Maximum allowed simultaneous workers provisioning" json:"maxConcurrentProvisioning"`
		MaxConcurrentRegistering  int  `toml:"maxConcurrentRegistering" default:"2" comment:"Maximum allowed simultaneous workers registering. -1 to disable registering on this hatchery" json:"maxConcurrentRegistering"`
		RegisterFrequency         int  `toml:"registerFrequency" default:"60" comment:"Check if some worker model have to be registered each n Seconds" json:"registerFrequency"`
		WorkerLogsOptions         struct {
			Graylog struct {
				Host       string `toml:"host" comment:"Example: thot.ovh.com" json:"host"`
				Port       int    `toml:"port" comment:"Example: 12202" json:"port"`
				Protocol   string `toml:"protocol" default:"tcp" comment:"tcp or udp" json:"protocol"`
				ExtraKey   string `toml:"extraKey" comment:"Example: X-OVH-TOKEN. You can use many keys: aaa,bbb" json:"extraKey"`
				ExtraValue string `toml:"extraValue" comment:"value for extraKey field. For many keys: valueaaa,valuebbb" json:"-"`
			} `toml:"graylog" json:"graylog"`
		} `toml:"workerLogsOptions" comment:"Worker Log Configuration" json:"workerLogsOptions"`
	} `toml:"provision" json:"provision"`
	LogOptions struct {
		SpawnOptions struct {
			ThresholdCritical int `toml:"thresholdCritical" default:"480" comment:"log critical if spawn take more than this value (in seconds)" json:"thresholdCritical"`
			ThresholdWarning  int `toml:"thresholdWarning" default:"360" comment:"log warning if spawn take more than this value (in seconds)" json:"thresholdWarning"`
		} `toml:"spawnOptions" json:"spawnOptions"`
	} `toml:"logOptions" comment:"Hatchery Log Configuration" json:"logOptions"`
}

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
	JobID        int64             `json:"job_id"`
	Requirements []sdk.Requirement `json:"requirements"`
	RegisterOnly bool              `json:"register_only"`
	HatcheryName string            `json:"hatchery_name"`
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
	InitHatchery() error
	SpawnWorker(ctx context.Context, spawnArgs SpawnArguments) error
	CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool

	WorkersStarted() []string
	Hatchery() *sdk.Hatchery
	Service() *sdk.Service
	CDSClient() cdsclient.Interface
	Configuration() CommonConfiguration

	Serve(ctx context.Context) error
	ServiceName() string
	Metrics() *Metrics
	PanicDumpDirectory() (string, error)
	PrivateKey() *rsa.PrivateKey
}

type InterfaceWithModels interface {
	Interface
	WorkersStartedByModel(model *sdk.Model) int
	ModelType() string
	NeedRegistration(model *sdk.Model) bool
	WorkerModelsEnabled() ([]sdk.Model, error)
}

type Metrics struct {
	Jobs               *stats.Int64Measure
	JobsSSE            *stats.Int64Measure
	SpawnedWorkers     *stats.Int64Measure
	PendingWorkers     *stats.Int64Measure
	RegisteringWorkers *stats.Int64Measure
	CheckingWorkers    *stats.Int64Measure
	WaitingWorkers     *stats.Int64Measure
	BuildingWorkers    *stats.Int64Measure
	DisabledWorkers    *stats.Int64Measure
}
