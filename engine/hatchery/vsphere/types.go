package vsphere

import (
	"sync"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/engine/service"
	"github.com/vmware/govmomi/vim25/mo"
)

// HatcheryConfiguration is the configuration for hatchery
type HatcheryConfiguration struct {
	service.HatcheryCommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration" json:"commonConfiguration"`
	VSphereUser                         string                     `mapstructure:"user" toml:"user" default:"" commented:"false" comment:"VSphere User" json:"user"`
	VSphereEndpoint                     string                     `mapstructure:"endpoint" toml:"endpoint" default:"" commented:"false" comment:"VShpere Endpoint, example:pcc-11-222-333-444.ovh.com" json:"endpoint"`
	VSpherePassword                     string                     `mapstructure:"password" toml:"password" default:"" commented:"false" comment:"VShpere Password" json:"-"`
	VSphereDatacenterString             string                     `mapstructure:"datacenterString" toml:"datacenterString" default:"" commented:"false" comment:"VSphere Datacenter" json:"datacenterString"`
	VSphereDatastoreString              string                     `mapstructure:"datastoreString" toml:"datastoreString" default:"" commented:"false" comment:"VSphere Datastore" json:"datastoreString"`
	VSphereNetworkString                string                     `mapstructure:"networkString" toml:"networkString" default:"" commented:"false" comment:"VShpere Network" json:"networkString"`
	VSphereCardName                     string                     `mapstructure:"cardName" toml:"cardName" default:"e1000" commented:"false" comment:"Name of the virtual ethernet card" json:"cardName"`
	IPRange                             string                     `mapstructure:"iprange" toml:"iprange" default:"" commented:"false" comment:"Optional. IP Range for spawned workers. \n Format: a.a.a.a/b,c.c.c.c/e \n Hatchery will use an IP from this range to create Virtual Machine (Fixed IP Attribute).\nIf not set, you have to set it in your worker model template" json:"iprange,omitempty"`
	Gateway                             string                     `mapstructure:"gateway" toml:"gateway" default:"" commented:"false" comment:"Optional. Gateway IP for spawned workers." json:"gateway,omitempty"`
	DNS                                 string                     `mapstructure:"dns" toml:"dns" default:"" commented:"false" comment:"Optional. DNS IP" json:"dns,omitempty"`
	SubnetMask                          string                     `mapstructure:"subnetMask" toml:"subnetMask" default:"255.255.255.0" commented:"false" comment:"Subnet Mask" json:"subnetMask"`
	WorkerTTL                           int                        `mapstructure:"workerTTL" toml:"workerTTL" default:"120" commented:"false" comment:"Worker TTL (minutes)" json:"workerTTL"`
	WorkerRegistrationTTL               int                        `mapstructure:"workerRegistrationTTL" toml:"workerRegistrationTTL" commented:"false" comment:"Worker Registration TTL (minutes)" json:"workerRegistrationTTL"`
	WorkerProvisioning                  []WorkerProvisioningConfig `mapstructure:"workerProvisioning" toml:"workerProvisioning" commented:"false" comment:"Worker Provisioning per model name" json:"workerProvisioning"`
}

type WorkerProvisioningConfig struct {
	ModelPath string `mapstructure:"modelPath" toml:"modelPath" json:"modelPath"`
	Number    int64  `mapstructure:"number" toml:"number" json:"number"`
}

// HatcheryVSphere spawns vm
type HatcheryVSphere struct {
	hatcheryCommon.Common
	Config               HatcheryConfiguration
	vSphereClient        VSphereClient
	IpAddressesMutex     sync.Mutex
	availableIPAddresses []string
	reservedIPAddresses  []string
	cacheVirtualMachines struct {
		mu   sync.Mutex
		list []mo.VirtualMachine
	}
	cacheTemplates struct {
		mu   sync.Mutex
		list []mo.VirtualMachine
	}
	cachePendingJobID struct {
		mu   sync.Mutex
		list []int64
	}
	cacheProvisioning struct {
		mu         sync.Mutex
		pending    []string
		restarting []string
	}
	cacheToDelete struct {
		mu   sync.Mutex
		list []string
	}
}
