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
	IPRange    string `mapstructure:"iprange" toml:"iprange" default:"" commented:"false" comment:"Optional. IP Range for spawned workers. \n Format: a.a.a.a/b,c.c.c.c/e \n Hatchery will use an IP from this range to create Virtual Machine (Fixed IP Attribute).\nIf not set, you have to set it in your worker model template" json:"iprange,omitempty"`
	Gateway    string `mapstructure:"gateway" toml:"gateway" default:"" commented:"false" comment:"Optional. Gateway IP for spawned workers." json:"gateway,omitempty"`
	DNS        string `mapstructure:"dns" toml:"dns" default:"" commented:"false" comment:"Optional. DNS IP" json:"dns,omitempty"`
	SubnetMask string `mapstructure:"subnetMask" toml:"subnetMask" default:"255.255.255.0" commented:"false" comment:"Subnet Mask" json:"subnetMask"`

	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `mapstructure:"workerTTL" toml:"workerTTL" default:"30" commented:"false" comment:"Worker TTL (minutes)" json:"workerTTL"`
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
