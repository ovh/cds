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
	hatchery.CommonConfiguration `toml:"commonConfiguration"`

	// Tenant openstack-tenant
	Tenant string `toml:"tenant" default:"" commented:"true" comment:"Openstack tenant (string)"`

	// User  openstack-user
	User string `toml:"user" default:"" commented:"true" comment:"Openstack User"`

	// Address  openstack-auth-endpoint
	Address string `toml:"address" default:"https://auth.cloud.ovh.net/v2.0" commented:"true" comment:"Opentack Auth Endpoint"`

	// Password openstack-password
	Password string `toml:"password" default:"" commented:"true" comment:"Openstack Password"`

	// Region openstack-region
	Region string `toml:"region" default:"" commented:"true" comment:"Openstack Region"`

	// NetworkString openstack-network
	NetworkString string `toml:"networkString" default:"Ext-Net" commented:"true" comment:"Hatchery will use this Network to spawn CDS Worker (Virtual Machine)."`

	// IPRange IP Range
	IPRange string `toml:"iprange" default:"" commented:"true" comment:"Facultative. IP Range for spawned workers. \n Format: a.a.a.a/b,c.c.c.c/e \n Hatchery will use an IP from this range to create Virtual Machine (Fixed IP Attribute).\nIf not set, it will get an address from the neutron service"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `toml:"workerTTL" default:"30" commented:"true" comment:"Worker TTL (minutes)"`

	// DisableCreateImage if true: hatchery does not create openstack image when a worker model is updated
	DisableCreateImage bool `toml:"disableCreateImage" default:"false" commented:"true" comment:"if true: hatchery does not create openstack image when a worker model is updated"`

	// CreateImageTimeout max wait for create an openstack image (in seconds)
	CreateImageTimeout int `toml:"createImageTimeout" default:"180" commented:"true" comment:"max wait for create an openstack image (in seconds)"`
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
