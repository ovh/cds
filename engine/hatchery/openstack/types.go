package openstack

import (
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/tenantnetworks"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration

	// Tenant openstack-tenant
	Tenant string `default:""`

	// User  openstack-user
	User string `default:""`

	// Address  openstack-auth-endpoint
	Address string `default:""`

	// Password openstack-password
	Password string `default:""`

	// Region openstack-region
	Region string `default:""`

	// NetworkString openstack-network
	NetworkString string `default:"Ext-Net"`

	// IPRange IP Range
	IPRange string `default:""`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `default:"30"`

	// DisableCreateImage if true: hatchery does not create openstack image when a worker model is updated
	DisableCreateImage bool `default:"false"`

	// CreateImageTimeout max wait for create an openstack image (in seconds)
	CreateImageTimeout int `default:"180"`
}

// HatcheryOpenstack spawns instances of worker model with type 'ISO'
// by startup up virtual machines on /cloud
type HatcheryOpenstack struct {
	Config          HatcheryConfiguration
	hatch           *sdk.Hatchery
	flavors         []flavors.Flavor
	networks        []tenantnetworks.Network
	images          []images.Image
	openstackClient *gophercloud.ServiceClient
	client          cdsclient.Interface

	networkID string // computed from networkString
}

type ipInfos struct {
	workerName     string
	dateLastBooked time.Time
}
