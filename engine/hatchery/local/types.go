package local

import (
	"os/exec"
	"sync"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

// HatcheryConfiguration is the configuration for local hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration"`
	Basedir                      string `mapstructure:"basedir" toml:"basedir" default:"/tmp" comment:"BaseDir for worker workspace"`
	NbProvision                  int    `mapstructure:"nbProvision" toml:"nbProvision" default:"1" comment:"Nb Workers to provision"`
}

// HatcheryLocal implements HatcheryMode interface for local usage
type HatcheryLocal struct {
	Config HatcheryConfiguration
	sync.Mutex
	hatch   *sdk.Hatchery
	workers map[string]workerCmd
	client  cdsclient.Interface
	os      string
	arch    string
}

type workerCmd struct {
	cmd     *exec.Cmd
	created time.Time
}
