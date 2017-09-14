package docker

import (
	"os/exec"
	"sync"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

// HatcheryConfiguration is the configuration for docker hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `toml:"commonConfiguration"`
	DockerAddHost                string `toml:"dockerAddHost" default:"" commented:"true" comment:"Start worker with a custom host-to-IP mapping (host:ip)"`
}

// HatcheryDocker spawns instances of worker model with type 'Docker'
// by directly using available docker daemon
type HatcheryDocker struct {
	Config HatcheryConfiguration
	sync.Mutex
	workers map[string]*exec.Cmd
	hatch   *sdk.Hatchery
	client  cdsclient.Interface
}
