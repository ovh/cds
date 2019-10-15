package local

import (
	"os/exec"
	"sync"
	"time"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

// HatcheryConfiguration is the configuration for local hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration" json:"commonConfiguration"`
	Basedir                      string `mapstructure:"basedir" toml:"basedir" default:"/var/lib/cds-engine" comment:"BaseDir for worker workspace" json:"basedir"`
	NbProvision                  int    `mapstructure:"nbProvision" toml:"nbProvision" default:"1" comment:"Nb Workers to provision" json:"nbProvision"`
}

// HatcheryLocal implements HatcheryMode interface for local usage
type HatcheryLocal struct {
	hatcheryCommon.Common
	Config HatcheryConfiguration
	sync.Mutex
	hatch            *sdk.Hatchery
	workers          map[string]workerCmd
	ModelLocal       sdk.Model
	BasedirDedicated string
}

type workerCmd struct {
	cmd     *exec.Cmd
	created time.Time
}
