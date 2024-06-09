package local

import (
	"context"
	"os/exec"
	"sync"
	"time"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/engine/service"
)

// HatcheryConfiguration is the configuration for local hatchery
type HatcheryConfiguration struct {
	service.HatcheryCommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration" json:"commonConfiguration"`
	Basedir                             string `mapstructure:"basedir" toml:"basedir" default:"/var/lib/cds-engine" comment:"BaseDir for worker workspace" json:"basedir"`
}

// HatcheryLocal implements HatcheryMode interface for local usage
type HatcheryLocal struct {
	hatcheryCommon.Common
	Config HatcheryConfiguration
	sync.Mutex
	workers           map[string]workerCmd
	LocalWorkerRunner LocalWorkerRunner
	// BasedirDedicated = basedir + hatchery.name
	// this directory contains the worker downloaded from api at startup
	BasedirDedicated string
}

type workerCmd struct {
	cmd     *exec.Cmd
	created time.Time
}

type LocalWorkerRunner interface {
	NewCmd(ctx context.Context, command string, args ...string) *exec.Cmd
}
