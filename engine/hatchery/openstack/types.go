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
	"github.com/ovh/cds/sdk/service"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration"`

	// Tenant openstack-tenant
	Tenant string `mapstructure:"tenant" toml:"tenant" default:"" commented:"false" comment:"Openstack tenant (string)"`

	// User  openstack-user
	User string `mapstructure:"user" toml:"user" default:"" commented:"false" comment:"Openstack User"`

	// Address  openstack-auth-endpoint
	Address string `mapstructure:"address" toml:"address" default:"https://auth.cloud.ovh.net/v2.0" commented:"false" comment:"Opentack Auth Endpoint"`

	// Password openstack-password
	Password string `mapstructure:"password" toml:"password" default:"" commented:"false" comment:"Openstack Password"`

	// Region openstack-region
	Region string `mapstructure:"region" toml:"region" default:"" commented:"false" comment:"Openstack Region"`

	// NetworkString openstack-network
	NetworkString string `mapstructure:"networkString" toml:"networkString" default:"Ext-Net" commented:"false" comment:"Hatchery will use this Network to spawn CDS Worker (Virtual Machine)."`

	// IPRange IP Range
	IPRange string `mapstructure:"iprange" toml:"iprange" default:"" commented:"false" comment:"Facultative. IP Range for spawned workers. \n Format: a.a.a.a/b,c.c.c.c/e \n Hatchery will use an IP from this range to create Virtual Machine (Fixed IP Attribute).\nIf not set, it will get an address from the neutron service"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `mapstructure:"workerTTL" toml:"workerTTL" default:"30" commented:"false" comment:"Worker TTL (minutes)"`

	// DisableCreateImage if true: hatchery does not create openstack image when a worker model is updated
	DisableCreateImage bool `mapstructure:"disableCreateImage" toml:"disableCreateImage" default:"false" commented:"false" comment:"if true: hatchery does not create openstack image when a worker model is updated"`

	// CreateImageTimeout max wait for create an openstack image (in seconds)
	CreateImageTimeout int `mapstructure:"createImageTimeout" toml:"createImageTimeout" default:"180" commented:"false" comment:"max wait for create an openstack image (in seconds)"`
}

// HatcheryOpenstack spawns instances of worker model with type 'ISO'
// by startup up virtual machines on /cloud
type HatcheryOpenstack struct {
	service.Common
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
