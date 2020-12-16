package vsphere

import (
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/engine/service"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	service.HatcheryCommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration" json:"commonConfiguration"`

	// User vsphere-user
	VSphereUser string `mapstructure:"user" toml:"user" default:"" commented:"false" comment:"VSphere User" json:"user"`

	// Endpoint vsphere-endpoint
	VSphereEndpoint string `mapstructure:"endpoint" toml:"endpoint" default:"" commented:"false" comment:"VShpere Endpoint, example:pcc-11-222-333-444.ovh.com" json:"endpoint"`

	// Password vsphere-password
	VSpherePassword string `mapstructure:"password" toml:"password" default:"" commented:"false" comment:"VShpere Password" json:"-"`

	// DatacenterString vsphere-datacenter
	VSphereDatacenterString string `mapstructure:"datacenterString" toml:"datacenterString" default:"" commented:"false" comment:"VSphere Datacenter" json:"datacenterString"`

	// DatastoreString vsphere-datastore
	VSphereDatastoreString string `mapstructure:"datastoreString" toml:"datastoreString" default:"" commented:"false" comment:"VSphere Datastore" json:"datastoreString"`

	// NetworkString vsphere-network VM Network
	VSphereNetworkString string `mapstructure:"networkString" toml:"networkString" default:"" commented:"false" comment:"VShpere Network" json:"networkString"`

	// CardName vsphere-ethernet-card Name of the virtual ethernet card
	VSphereCardName string `mapstructure:"cardName" toml:"cardName" default:"e1000" commented:"false" comment:"Name of the virtual ethernet card" json:"cardName"`

	// IPRange IP Range
	IPRange string `mapstructure:"iprange" toml:"iprange" default:"" commented:"false" comment:"Facultative. IP Range for spawned workers. \n Format: a.a.a.a/b,c.c.c.c/e \n Hatchery will use an IP from this range to create Virtual Machine (Fixed IP Attribute).\nIf not set, it will get an address from the neutron service" json:"iprange,omitempty"`

	Gateway string `mapstructure:"gateway" toml:"gateway" default:"" commented:"false" comment:"Facultative. Gateway IP for spawned workers." json:"gateway,omitempty"`

	DNS string `mapstructure:"dns" toml:"dns" default:"" commented:"false" comment:"Facultative. DNS IP" json:"dns,omitempty"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `mapstructure:"workerTTL" toml:"workerTTL" default:"30" commented:"false" comment:"Worker TTL (minutes)" json:"workerTTL"`

	// DisableCreateImage if true: hatchery does not create vsphere image when a worker model is updated
	DisableCreateImage bool `mapstructure:"disableCreateImage" toml:"disableCreateImage" default:"false" commented:"false" comment:"if true: hatchery does not create vsphere image when a worker model is updated" json:"disableCreateImage"`

	// CreateImageTimeout max wait for create a vsphere image (in seconds)
	CreateImageTimeout int `mapstructure:"createImageTimeout" toml:"createImageTimeout" default:"180" commented:"false" comment:"max wait for create a vsphere image (in seconds)" json:"createImageTimeout"`
}

// HatcheryVSphere spawns vm
type HatcheryVSphere struct {
	hatcheryCommon.Common
	Config     HatcheryConfiguration
	images     []string
	datacenter *object.Datacenter
	finder     *find.Finder
	network    object.NetworkReference
	vclient    *govmomi.Client
}

type ipInfos struct {
	workerName     string
	dateLastBooked time.Time
}
