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

	// MarathonURL "marathon-api"
	MarathonURL string `default:"https://lb.gra-1.containers.ovh.net/marathon/yourstack/" commented:"true" comment:"URL of your marathon"`

	// MarathonID "marathon-id"
	MarathonIDPrefix string `default:"/cds/workers" commented:"true" comment:"Prefix of id for workers spawn on marathon. Enter 'workers' to have id as: '/workers/a-worker'"`

	// MarathonUser "marathon-user"
	MarathonUser string `default:"" commented:"true" comment:"Marathon Username, used to call Marathon URL"`

	// MarathonPassword "marathon-password"
	MarathonPassword string `default:"" commented:"true" comment:"Marathon Password, you need a marathon User to use it"`

	// MarathonLabelsStr "marathon-labels"
	MarathonLabels string `default:"" commented:"true" comment:"Use this option if you want to add labels on workers spawned by this hatchery.\n Format: MarathonLabels = \"A_LABEL=value-of-label\""`

	// DefaultMemory Worker default memory
	DefaultMemory int `default:"1024" commented:"true" comment:"Worker default memory in Mo"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `default:"10" commented:"true" comment:"Worker TTL (minutes)"`

	// WorkerSpawnTimeout Worker Timeout Spawning (seconds)
	WorkerSpawnTimeout int `default:"120" commented:"true" comment:"Worker Timeout Spawning (seconds)"`
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
