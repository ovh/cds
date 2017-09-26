package local

import (
	"os/exec"
	"sync"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

// HatcheryConfiguration is the configuration for local hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `toml:"commonConfiguration"`
	Basedir                      string `toml:"basedir" default:"/tmp" comment:"BaseDir for worker workspace"`
	NbProvision                  int    `toml:"nbProvision" default:"1" comment:"Nb Workers to provision"`
}

// HatcheryLocal implements HatcheryMode interface for local usage
type HatcheryLocal struct {
	Config HatcheryConfiguration
	sync.Mutex
	hatch   *sdk.Hatchery
	workers map[string]*exec.Cmd
	client  cdsclient.Interface
	os      string
	arch    string
}
