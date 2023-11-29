package swarm

import (
	docker "github.com/moby/moby/client"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/engine/service"
)

const (
	LabelHatchery           = "hatchery"
	LabelWorkerName         = "worker_name"
	LabelServiceName        = "service_name"
	LabelWorkerRequirements = "worker_requirements"
	LabelWorkerModelPath    = "worker_model_path"
	LabelJobID              = "job_id"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	service.HatcheryCommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration"`

	// MaxContainers
	MaxContainers int `mapstructure:"maxContainers" toml:"maxContainers" default:"10" commented:"false" comment:"Max Containers on Host managed by this Hatchery" json:"maxContainers"`

	// DefaultMemory Worker default memory
	DefaultMemory     int  `mapstructure:"defaultMemory" toml:"defaultMemory" default:"1024" commented:"false" comment:"Worker default memory in Mo" json:"defaultMemory"`
	DisableMemorySwap bool `mapstructure:"disableMemorySwap" toml:"disableMemorySwap" default:"false" commented:"true" comment:"Set to true to disable memory swap" json:"disableMemorySwap"`

	// DockerOpts Docker options
	DockerOpts string `mapstructure:"dockerOpts" toml:"dockerOpts" default:"" commented:"true" comment:"Docker Options. --add-host and --privileged supported. Example: dockerOpts=\"--add-host=myhost:x.x.x.x,myhost2:y.y.y.y --privileged\"" json:"dockerOpts,omitempty"`

	// TODO refactor DockerOpts globally: issue #4594
	DisableDockerOptsOnRequirements bool `mapstructure:"disableDockerOptsOnRequirements" toml:"disableDockerOptsOnRequirements" default:"" commented:"true" comment:"disable dockerOpts on requirements"`

	// NetworkEnableIPv6 if true: set ipv6 to true
	NetworkEnableIPv6 bool `mapstructure:"networkEnableIPv6" toml:"networkEnableIPv6" default:"false" commented:"false" comment:"if true: hatchery creates private network between services with ipv6 enabled" json:"networkEnableIPv6"`

	DockerEngines map[string]DockerEngineConfiguration `mapstructure:"dockerEngines" toml:"dockerEngines" comment:"List of Docker Engines" json:"dockerEngines,omitempty"`

	RegistryCredentials []RegistryCredential `mapstructure:"registryCredentials" toml:"registryCredentials" commented:"true" comment:"List of Docker registry credentials" json:"-"`

	WorkerMetricsRefreshDelay int64 `toml:"workerMetricsRefreshDelay" json:"workerMetricsRefreshDelay" commented:"true" comment:"Interval to compute worker metrics (in seconds), set to 0 will disable worker metrics."`
}

// HatcherySwarm is a hatchery which can be connected to a remote to a docker remote api
type HatcherySwarm struct {
	hatcheryCommon.Common
	Config        HatcheryConfiguration
	dockerClients map[string]*dockerClient
	workerMetrics struct {
		CPU               *stats.Float64Measure
		CPURequest        *stats.Float64Measure
		Memory            *stats.Int64Measure
		MemoryRequest     *stats.Int64Measure
		CPUView           *view.View
		CPURequestView    *view.View
		MemoryView        *view.View
		MemoryRequestView *view.View
	}
}

type dockerClient struct {
	docker.Client
	MaxContainers int
	name          string
}

// DockerEngineConfiguration is a configuration to be able to connect to a docker engine
type DockerEngineConfiguration struct {
	Host                  string `mapstructure:"host" toml:"host" comment:"DOCKER_HOST" json:"host"`                                                                        // DOCKER_HOST
	CertPath              string `mapstructure:"certPath" toml:"certPath" comment:"DOCKER_CERT_PATH" json:"-"`                                                              // DOCKER_CERT_PATH
	InsecureSkipTLSVerify bool   `mapstructure:"insecureSkipTLSVerify" toml:"insecureSkipTLSVerify" comment:"DOCKER_INSECURE_SKIP_TLS_VERIFY" json:"insecureSkipTLSVerify"` // !DOCKER_TLS_VERIFY
	TLSCAPEM              string `mapstructure:"TLSCAPEM" toml:"TLSCAPEM" comment:"content of your ca.pem" json:"-"`
	TLSCERTPEM            string `mapstructure:"TLSCERTPEM" toml:"TLSCERTPEM" comment:"content of your cert.pem" json:"-"`
	TLSKEYPEM             string `mapstructure:"TLSKEYPEM" toml:"TLSKEYPEM" comment:"content of your key.pem" json:"-"`
	APIVersion            string `mapstructure:"APIVersion" toml:"APIVersion" default:"1.41" comment:"DOCKER_API_VERSION" json:"APIVersion"` // DOCKER_API_VERSION
	MaxContainers         int    `mapstructure:"maxContainers" toml:"maxContainers" default:"10" commented:"false" comment:"Max Containers on Host managed by this Hatchery" json:"maxContainers"`
}

type RegistryCredential struct {
	Domain   string `mapstructure:"domain" default:"docker.io" commented:"true" toml:"domain" json:"-"`
	Username string `mapstructure:"username" commented:"true" toml:"username" json:"-"`
	Password string `mapstructure:"password" commented:"true" toml:"password" json:"-"`
}

type WorkerMetricsResource struct {
	WorkerName    string
	JobID         int64
	Node          string
	Name          string
	CPU           float64
	CPURequest    float64
	Memory        int64
	MemoryRequest int64
}
