package vsphere

import (
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration"`

	// User vsphere-user
	VSphereUser string `mapstructure:"user" toml:"user" default:"" commented:"true" comment:"VSphere User"`

	// Endpoint vsphere-endpoint
	VSphereEndpoint string `mapstructure:"endpoint" toml:"endpoint" default:"" commented:"true" comment:"VShpere Endpoint, example:pcc-11-222-333-444.ovh.com"`

	// Password vsphere-password
	VSpherePassword string `mapstructure:"password" toml:"password" default:"" commented:"true" comment:"VShpere Password"`

	// DatacenterString vsphere-datacenter
	VSphereDatacenterString string `mapstructure:"datacenterString" toml:"datacenterString" default:"" commented:"true" comment:"VSphere Datacenter"`

	// DatastoreString vsphere-datastore
	VSphereDatastoreString string `mapstructure:"datastoreString" toml:"datastoreString" default:"" commented:"true" comment:"VSphere Datastore"`

	// NetworkString vsphere-network VM Network
	VSphereNetworkString string `mapstructure:"networkString" toml:"networkString" default:"" commented:"true" comment:"VShpere Network"`

	// CardName vsphere-ethernet-card Name of the virtual ethernet card
	VSphereCardName string `mapstructure:"cardName" toml:"cardName" default:"e1000" commented:"true" comment:"Name of the virtual ethernet card"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `mapstructure:"workerTTL" toml:"workerTTL" default:"30" commented:"true" comment:"Worker TTL (minutes)"`

	// DisableCreateImage if true: hatchery does not create vsphere image when a worker model is updated
	DisableCreateImage bool `mapstructure:"disableCreateImage" toml:"disableCreateImage" default:"false" commented:"true" comment:"if true: hatchery does not create vsphere image when a worker model is updated"`

	// CreateImageTimeout max wait for create a vsphere image (in seconds)
	CreateImageTimeout int `mapstructure:"createImageTimeout" toml:"createImageTimeout" default:"180" commented:"true" comment:"max wait for create a vsphere image (in seconds)"`
}

// HatcheryVSphere spawns vm
type HatcheryVSphere struct {
	Config     HatcheryConfiguration
	hatch      *sdk.Hatchery
	images     []string
	datacenter *object.Datacenter
	finder     *find.Finder
	network    object.NetworkReference
	vclient    *govmomi.Client
	client     cdsclient.Interface

	// User provided parameters
	endpoint           string
	user               string
	password           string
	host               string
	datacenterString   string
	datastoreString    string
	networkString      string
	cardName           string
	workerTTL          int
	disableCreateImage bool
	createImageTimeout int
}
