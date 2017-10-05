package marathon

import (
	"github.com/gambol99/go-marathon"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration"`

	// MarathonURL "marathon-api"
	MarathonURL string `mapstructure:"url" toml:"url" default:"http://10.241.1.71:8080,10.241.1.72:8080,10.241.1.73:8080" commented:"true" comment:"URL of your marathon"`

	// MarathonID "marathon-id"
	MarathonIDPrefix string `mapstructure:"idPrefix" toml:"idPrefix" default:"/cds/workers" commented:"true" comment:"Prefix of id for workers spawn on marathon. Enter 'workers' to have id as: '/workers/a-worker'"`

	// MarathonUser "marathon-user"
	MarathonUser string `mapstructure:"user" toml:"user" default:"" commented:"true" comment:"Marathon Username, used to call Marathon URL"`

	// MarathonPassword "marathon-password"
	MarathonPassword string `mapstructure:"password" toml:"password" default:"" commented:"true" comment:"Marathon Password, you need a marathon User to use it"`

	// MarathonLabelsStr "marathon-labels"
	MarathonLabels string `mapstructure:"labels" toml:"labels" default:"" commented:"true" comment:"Use this option if you want to add labels on workers spawned by this hatchery.\n Format: MarathonLabels = \"A_LABEL=value-of-label\""`

	// DefaultMemory Worker default memory
	DefaultMemory int `mapstructure:"defaultMemory" toml:"defaultMemory" default:"1024" commented:"true" comment:"Worker default memory in Mo"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `mapstructure:"workerTTL" toml:"workerTTL" default:"10" commented:"true" comment:"Worker TTL (minutes)"`

	// WorkerSpawnTimeout Worker Timeout Spawning (seconds)
	WorkerSpawnTimeout int `mapstructure:"workerSpawnTimeout" toml:"workerSpawnTimeout" default:"120" commented:"true" comment:"Worker Timeout Spawning (seconds)"`
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
