package marathon

import (
	"github.com/gambol99/go-marathon"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration

	// MarathonHost "marathon-host"
	MarathonHost string `default:""`

	// MarathonID "marathon-id"
	MarathonID string `default:""`

	// MarathonUser "marathon-user"
	MarathonUser string `default:""`

	// MarathonPassword "marathon-password"
	MarathonPassword string `default:""`

	// MarathonLabelsStr "marathon-labels"
	MarathonLabelsString string `default:""`

	// DefaultMemory Worker default memory
	DefaultMemory int `default:"1024"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `default:"10"`

	// WorkerSpawnTimeout , "Worker Timeout Spawning (seconds)
	WorkerSpawnTimeout int `default:"120"`
}

// HatcheryMarathon implements HatcheryMode interface for mesos mode
type HatcheryMarathon struct {
	Config HatcheryConfiguration
	hatch  *sdk.Hatchery
	token  string

	marathonClient marathon.Marathon
	client         cdsclient.Interface

	marathonLabels map[string]string
}
