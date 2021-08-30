package marathon

import (
	"github.com/gambol99/go-marathon"
	"github.com/ovh/cds/engine/service"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	service.HatcheryCommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration" json:"commonConfiguration"`

	// MarathonURL "marathon-api"
	MarathonURL string `mapstructure:"url" toml:"url" default:"http://1.1.1.1:8080,1.1.1.2:8080,1.1.1.3:8080" commented:"false" comment:"URL of your marathon" json:"url"`

	// MarathonID "marathon-id"
	MarathonIDPrefix string `mapstructure:"idPrefix" toml:"idPrefix" default:"/cds/workers" commented:"false" comment:"Prefix of id for workers spawn on marathon. Enter 'workers' to have id as: '/workers/a-worker'" json:"idPrefix"`

	// MarathonUser "marathon-user"
	MarathonUser string `mapstructure:"user" toml:"user" default:"" commented:"false" comment:"Marathon Username, used to call Marathon URL" json:"user"`

	// MarathonPassword "marathon-password"
	MarathonPassword string `mapstructure:"password" toml:"password" default:"" commented:"false" comment:"Marathon Password, you need a marathon User to use it" json:"-"`

	// MarathonLabelsStr "marathon-labels"
	MarathonLabels string `mapstructure:"labels" toml:"labels" default:"" commented:"false" comment:"Use this option if you want to add labels on workers spawned by this hatchery.\n Format: MarathonLabels = \"A_LABEL=value-of-label,B_LABEL=value-of-label-b\"" json:"labels"`

	// DefaultCPUs
	DefaultCPUs float64 `mapstructure:"defaultCPUs" toml:"defaultCPUs" default:"1" commented:"false" comment:"Worker default CPUs count" json:"defaultCPUs"`

	// DefaultMemory Worker default memory
	DefaultMemory int `mapstructure:"defaultMemory" toml:"defaultMemory" default:"1024" commented:"false" comment:"Worker default memory in Mo" json:"defaultMemory"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `mapstructure:"workerTTL" toml:"workerTTL" default:"10" commented:"false" comment:"Worker TTL (minutes)" json:"workerTTL"`

	// WorkerSpawnTimeout Worker Timeout Spawning (seconds)
	WorkerSpawnTimeout int `mapstructure:"workerSpawnTimeout" toml:"workerSpawnTimeout" default:"120" commented:"false" comment:"Worker Timeout Spawning (seconds)" json:"workerSpawnTimeout"`

	// MarathonApplicationURIs will set "uris" value for each Application
	MarathonApplicationURIs []string `mapstructure:"applicationURIs" toml:"applicationURIs" commented:"true" comment:"Use this option if you want to add uris on workers spawned by this hatchery." json:"-"`
}

// HatcheryMarathon implements HatcheryMode interface for mesos mode
type HatcheryMarathon struct {
	hatcheryCommon.Common
	Config         HatcheryConfiguration
	marathonClient marathon.Marathon
	marathonLabels map[string]string
}
