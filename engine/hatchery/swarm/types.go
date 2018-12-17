package swarm

import (
	docker "github.com/docker/docker/client"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration"`

	// RatioService Percent reserved for spawning worker with service requirement
	RatioService int `mapstructure:"ratioService" toml:"ratioService" default:"75" commented:"false" comment:"Percent reserved for spawning worker with service requirement" json:"ratioService"`

	// MaxContainers
	MaxContainers int `mapstructure:"maxContainers" toml:"maxContainers" default:"10" commented:"false" comment:"Max Containers on Host managed by this Hatchery" json:"maxContainers"`

	// DefaultMemory Worker default memory
	DefaultMemory int `mapstructure:"defaultMemory" toml:"defaultMemory" default:"1024" commented:"false" comment:"Worker default memory in Mo" json:"defaultMemory"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `mapstructure:"workerTTL" toml:"workerTTL" default:"10" commented:"false" comment:"Worker TTL (minutes)" json:"workerTTL"`

	// DockerOpts Docker options
	DockerOpts string `mapstructure:"dockerOpts" toml:"dockerOpts" default:"" commented:"true" comment:"Docker Options. --add-host and --privileged supported. Example: dockerOpts=\"--add-host=myhost:x.x.x.x,myhost2:y.y.y.y --privileged\"" json:"dockerOpts,omitempty"`

	// NetworkEnableIPv6 if true: set ipv6 to true
	NetworkEnableIPv6 bool `mapstructure:"networkEnableIPv6" toml:"networkEnableIPv6" default:"false" commented:"false" comment:"if true: hatchery creates private network between services with ipv6 enabled" json:"networkEnableIPv6"`

	DockerEngines map[string]DockerEngineConfiguration `mapstructure:"dockerEngines" toml:"dockerEngines" comment:"List of Docker Engines" json:"dockerEngines,omitempty"`
}

// HatcherySwarm is a hatchery which can be connected to a remote to a docker remote api
type HatcherySwarm struct {
	hatcheryCommon.Common
	Config        HatcheryConfiguration
	hatch         *sdk.Hatchery
	dockerClients map[string]*dockerClient
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
	APIVersion            string `mapstructure:"APIVersion" toml:"APIVersion" comment:"DOCKER_API_VERSION" json:"APIVersion"` // DOCKER_API_VERSION
	MaxContainers         int    `mapstructure:"maxContainers" toml:"maxContainers" default:"10" commented:"false" comment:"Max Containers on Host managed by this Hatchery" json:"maxContainers"`
}
