package swarm

import (
	docker "github.com/docker/docker/client"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration"`

	// RatioService Percent reserved for spawning worker with service requirement
	RatioService int `mapstructure:"ratioService" toml:"ratioService" default:"75" commented:"false" comment:"Percent reserved for spawning worker with service requirement"`

	// MaxContainers
	MaxContainers int `mapstructure:"maxContainers" toml:"maxContainers" default:"10" commented:"false" comment:"Max Containers on Host managed by this Hatchery"`

	// DefaultMemory Worker default memory
	DefaultMemory int `mapstructure:"defaultMemory" toml:"defaultMemory" default:"1024" commented:"false" comment:"Worker default memory in Mo"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `mapstructure:"workerTTL" toml:"workerTTL" default:"10" commented:"false" comment:"Worker TTL (minutes)"`
}

// HatcherySwarm is a hatchery which can be connected to a remote to a docker remote api
type HatcherySwarm struct {
	service.Common
	Config       HatcheryConfiguration
	hatch        *sdk.Hatchery
	dockerClient *docker.Client
}
