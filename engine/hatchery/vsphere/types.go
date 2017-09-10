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
	hatchery.CommonConfiguration

	// User vsphere-user
	VSphereUser string `default:""`

	// Endpoint vsphere-endpoint
	VSphereEndpoint string `default:""`

	// Password vsphere-password
	VSpherePassword string `default:""`

	// DatacenterString vsphere-datacenter
	VSphereDatacenterString string `default:""`

	// DatastoreString vsphere-datastore
	VSphereDatastoreString string `default:""`

	// NetworkString vsphere-network VM Network
	VSphereNetworkString string `default:""`

	// CardName vsphere-ethernet-card Name of the virtual ethernet card
	VSphereCardName string `default:"e1000"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `default:"30"`

	// DisableCreateImage if true: hatchery does not create vsphere image when a worker model is updated
	DisableCreateImage bool `default:"false"`

	// CreateImageTimeout max wait for create a vsphere image (in seconds)
	CreateImageTimeout int `default:"180"`
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
