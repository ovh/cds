package hatchery

import (
	"context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

// CommonConfiguration is the base configuration for all hatcheries
type CommonConfiguration struct {
	Name string `toml:"name" default:"" comment:"Name of Hatchery"`
	HTTP struct {
		Addr string `toml:"addr" default:"" commented:"true" comment:"Listen address without port, example: 127.0.0.1"`
		Port int    `toml:"port" default:"8086"`
	} `toml:"http" comment:"######################\n CDS Hatchery HTTP Configuration \n######################"`
	URL string `toml:"url" default:"http://localhost:8086" comment:"URL of this Hatchery"`
	API struct {
		HTTP struct {
			URL      string `toml:"url" default:"http://localhost:8081" commented:"true" comment:"CDS API URL"`
			Insecure bool   `toml:"insecure" default:"false" commented:"true" comment:"sslInsecureSkipVerify, set to true if you use a self-signed SSL on CDS API"`
		} `toml:"http"`
		GRPC struct {
			URL      string `toml:"url" default:"http://localhost:8082" commented:"true"`
			Insecure bool   `toml:"insecure" default:"false" commented:"true" comment:"sslInsecureSkipVerify, set to true if you use a self-signed SSL on CDS API"`
		} `toml:"grpc"`
		Token                string `toml:"token" default:"" comment:"CDS Token to reach CDS API. See https://ovh.github.io/cds/advanced/advanced.worker.token/ "`
		RequestTimeout       int    `toml:"requestTimeout" default:"10" comment:"Request CDS API: timeout in seconds"`
		MaxHeartbeatFailures int    `toml:"maxHeartbeatFailures" default:"10" comment:"Maximum allowed consecutives failures on heatbeat routine"`
	} `toml:"api"`
	Provision struct {
		Disabled                  bool `toml:"disabled" default:"false" comment:"Disabled provisioning. Format:true or false"`
		Frequency                 int  `toml:"frequency" default:"30" comment:"Check provisioning each n Seconds"`
		MaxWorker                 int  `toml:"maxWorker" default:"10" comment:"Maximum allowed simultaneous workers"`
		MaxConcurrentProvisioning int  `toml:"maxConcurrentProvisioning" default:"10" comment:"Maximum allowed simultaneous workers provisioning"`
		GraceTimeQueued           int  `toml:"graceTimeQueued" default:"4" comment:"if worker is queued less than this value (seconds), hatchery does not take care of it"`
		RegisterFrequency         int  `toml:"registerFrequency" default:"60" comment:"Check if some worker model have to be registered each n Seconds"`
		WorkerLogsOptions         struct {
			Graylog struct {
				Host       string `toml:"host" comment:"Example: thot.ovh.com"`
				Port       int    `toml:"port" comment:"Example: 12202"`
				Protocol   string `toml:"protocol" default:"tcp" comment:"tcp or udp"`
				ExtraKey   string `toml:"extraKey" comment:"Example: X-OVH-TOKEN. You can use many keys: aaa,bbb"`
				ExtraValue string `toml:"extraValue" comment:"value for extraKey field. For many keys: valueaaa,valuebbb"`
			} `toml:"graylog"`
		} `toml:"workerLogsOptions" comment:"Worker Log Configuration"`
	} `toml:"provision"`
	LogOptions struct {
		SpawnOptions struct {
			ThresholdCritical int `toml:"thresholdCritical" default:"480" comment:"log critical if spawn take more than this value (in seconds)"`
			ThresholdWarning  int `toml:"thresholdWarning" default:"360" comment:"log warning if spawn take more than this value (in seconds)"`
		} `toml:"spawnOptions"`
	} `toml:"logOptions" comment:"Hatchery Log Configuration"`
}

// SpawnArguments contains arguments to func SpawnWorker
type SpawnArguments struct {
	Model         sdk.Model
	IsWorkflowJob bool
	JobID         int64
	Requirements  []sdk.Requirement
	RegisterOnly  bool
	LogInfo       string
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
	Init() error
	SpawnWorker(ctx context.Context, spawnArgs SpawnArguments) (string, error)
	CanSpawn(model *sdk.Model, jobID int64, requirements []sdk.Requirement) bool
	WorkersStartedByModel(model *sdk.Model) int
	WorkersStarted() []string
	Hatchery() *sdk.Hatchery
	CDSClient() cdsclient.Interface
	Configuration() CommonConfiguration
	ModelType() string
	NeedRegistration(model *sdk.Model) bool
	ID() int64
	Serve(ctx context.Context) error
	IsInitialized() bool
	SetInitialized()
	ServiceName() string
}
