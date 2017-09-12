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
	hatchery.CommonConfiguration

	// BaseDir for worker workspace
	Basedir string `default:"/tmp"`
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
