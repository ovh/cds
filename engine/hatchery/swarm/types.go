package swarm

import (
	"github.com/fsouza/go-dockerclient"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `toml:"commonConfiguration"`

	// RatioService Percent reserved for spwaning worker with service requirement
	RatioService int `toml:"ratioService" default:"75" commented:"true" comment:"Percent reserved for spwaning worker with service requirement"`

	// MaxContainers
	MaxContainers int `toml:"maxContainers" default:"10" commented:"true" comment:"Max Containers on Host managed by this Hatchery"`

	// DefaultMemory Worker default memory
	DefaultMemory int `toml:"defaultMemory" default:"1024" commented:"true" comment:"Worker default memory in Mo"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `toml:"workerTTL" default:"10" commented:"true" comment:"Worker TTL (minutes)"`
}

// HatcherySwarm is a hatchery which can be connected to a remote to a docker remote api
type HatcherySwarm struct {
	Config       HatcheryConfiguration
	hatch        *sdk.Hatchery
	dockerClient *docker.Client
	client       cdsclient.Interface
}
