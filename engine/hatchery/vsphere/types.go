package vsphere

import (
	"sync"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/engine/service"
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
	WorkerProvisioningInterval          int                        `mapstructure:"workerProvisioningInterval" toml:"workerProvisioningInterval" commented:"true" comment:"Worker Provisioning interval (seconds)" json:"workerProvisioningInterval"`
	WorkerProvisioningPoolSize          int                        `mapstructure:"workerProvisioningPoolSize" toml:"workerProvisioningPoolSize" commented:"true" comment:"Worker Provisioning pool size" json:"workerProvisioningPoolSize"`
	WorkerProvisioning                  []WorkerProvisioningConfig `mapstructure:"workerProvisioning" toml:"workerProvisioning" commented:"true" comment:"Worker Provisioning per model name" json:"workerProvisioning"`
	GuestCredentials                    []GuestCredential          `mapstructure:"guestCredentials" toml:"guestCredentials" commented:"true" comment:"List of Guest credentials" json:"-"`
	DefaultWorkerModelsV2               []DefaultWorkerModelsV2    `mapstructure:"defaultWorkerModelsV2" toml:"defaultWorkerModelsV2" commented:"true" comment:"List of default worker models v2 for declared binaries - used by workflow v1" json:"-"`
}

type WorkerProvisioningConfig struct {
	// ModelPath is the CDS worker model name, not the model from the VMWare point of view. It's used only by CDS Worker Model v1
	ModelPath string `mapstructure:"modelPath" default:"my/model" commented:"true" toml:"modelPath" json:"modelPath"`

	// ModelVMWare is the model from the VMWare point of view. It's used only by CDS Worker Model v2
	ModelVMWare string `mapstructure:"modelVMWare" default:"debian12" commented:"true" toml:"modelVMWare" json:"modelVMWare"`

	// Number of VM to provision for the current model
	Number int `mapstructure:"number" commented:"true" toml:"number" json:"number"`
}

type GuestCredential struct {
	// ModelPath is the CDS worker model name, it's used only by CDS Worker Model v1
	ModelPath string `mapstructure:"modelPath" default:"my/model" commented:"true" toml:"modelPath" json:"-"`

	// ModelVMWare is the model from the VMWare point of view. It's used only by CDS Worker Model v2
	ModelVMWare string `mapstructure:"modelVMWare" default:"debian12" commented:"true" toml:"modelVMWare" json:"-"`

	Username string `mapstructure:"username" commented:"true" toml:"username" json:"-"`
	Password string `mapstructure:"password" commented:"true" toml:"password" json:"-"`
}

// this is used to run worker model v2 in a job v1
type DefaultWorkerModelsV2 struct {
	WorkerModelV2 string   `mapstructure:"workerModelV2" default:"" commented:"true" toml:"workerModelV2" json:"workerModelV2"`
	Binaries      []string `mapstructure:"binaries" toml:"binaries" default:"" commented:"true" comment:"If one binary is matching this list, the default model associated is used." json:"binaries"`
}

// HatcheryVSphere spawns vm
type HatcheryVSphere struct {
	hatcheryCommon.Common
	Config               HatcheryConfiguration
	vSphereClient        VSphereClient
	IpAddressesMutex     sync.Mutex
	availableIPAddresses []string
	reservedIPAddresses  []string
	cachePendingJobID    struct {
		mu   sync.Mutex
		list []string
	}
	cacheProvisioning struct {
		mu      sync.Mutex
		pending []string
		using   []string
	}
	cacheToDelete struct {
		mu   sync.Mutex
		list []string
	}
}
