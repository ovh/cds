package microvm

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/engine/service"
)

const (
	LABEL_HATCHERY_NAME = "CDS_HATCHERY_NAME"
	LABEL_WORKER_NAME   = "CDS_WORKER_NAME"
)

// microVMResource is the MicroVM custom resource of the platform, used through
// the dynamic client to keep the hatchery free of any platform Go dependency.
var microVMResource = schema.GroupVersionResource{Group: "microvm.poc", Version: "v1alpha1", Resource: "microvms"}

const (
	microVMAPIVersion = "microvm.poc/v1alpha1"
	microVMKind       = "MicroVM"

	phasePending   = "Pending"
	phaseSucceeded = "Succeeded"
	phaseFailed    = "Failed"

	conditionQuotaExceeded = "QuotaExceeded"

	labelGuestFiles = "microvm.poc/guest-files"
)

// HatcheryConfiguration is the configuration for microvm hatchery
type HatcheryConfiguration struct {
	service.HatcheryCommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration" json:"commonConfiguration"`
	// KubernetesConfigFile Kubernetes config file in yaml, in-cluster configuration when empty
	KubernetesConfigFile string `mapstructure:"kubernetesConfigFile" toml:"kubernetesConfigFile" default:"" commented:"true" comment:"Kubernetes config file in yaml. The in-cluster configuration is used when empty" json:"kubernetesConfigFile"`
	// Namespace is the kubernetes namespace in which MicroVMs are created
	Namespace string `mapstructure:"namespace" toml:"namespace" default:"cds" commented:"false" comment:"Kubernetes namespace in which MicroVMs and their secrets are created" json:"namespace"`
	// EgressPolicy is the MicroVMEgressPolicy applied to every worker
	EgressPolicy string `mapstructure:"egressPolicy" toml:"egressPolicy" default:"" commented:"false" comment:"Name of the MicroVMEgressPolicy (same namespace) applied to the workers. It must allow the CDS API and CDN (HTTP and TCP). The platform default policy applies when empty" json:"egressPolicy"`
	// DebugSSHAuthorizedKeys enables the admin network of the workers with these root keys
	DebugSSHAuthorizedKeys []string `mapstructure:"debugSSHAuthorizedKeys" toml:"debugSSHAuthorizedKeys" default:"" commented:"true" comment:"Debugging aid: SSH public keys authorized as root on every worker. When set, the admin network of the workers is enabled (SSH from the bastion). Leave empty in normal use" json:"-"`
	// DefaultVCPUs used when the worker model does not set vcpus
	DefaultVCPUs int `mapstructure:"defaultVCPUs" toml:"defaultVCPUs" default:"2" commented:"false" comment:"Worker vCPUs when the worker model does not set them" json:"defaultVCPUs"`
	// DefaultMemory used when the worker model does not set memory
	DefaultMemory int `mapstructure:"defaultMemory" toml:"defaultMemory" default:"2048" commented:"false" comment:"Worker memory in MiB when the worker model does not set it" json:"defaultMemory"`
	// MaxVCPUs a worker model may ask for
	MaxVCPUs int `mapstructure:"maxVCPUs" toml:"maxVCPUs" default:"8" commented:"false" comment:"Maximum vCPUs of a worker. Jobs whose model asks for more are not taken" json:"maxVCPUs"`
	// MaxMemory a worker model may ask for
	MaxMemory int `mapstructure:"maxMemory" toml:"maxMemory" default:"16384" commented:"false" comment:"Maximum memory in MiB of a worker. Jobs whose model asks for more are not taken" json:"maxMemory"`
	// RegistrationTimeout delay after which a booted MicroVM without a registered worker is deleted
	RegistrationTimeout int `mapstructure:"registrationTimeout" toml:"registrationTimeout" default:"600" commented:"true" comment:"Delete a MicroVM whose worker is not registered after this delay (seconds), counted from the boot, or from the creation while not booted" json:"registrationTimeout"`
	// QuotaPendingTimeout delay after which a MicroVM refused by the namespace quota is deleted
	QuotaPendingTimeout int `mapstructure:"quotaPendingTimeout" toml:"quotaPendingTimeout" default:"1800" commented:"true" comment:"Delete a MicroVM still refused by the namespace quota after this delay (seconds). No new MicroVM is created while one is refused" json:"quotaPendingTimeout"`
	// KillAwolWorkersInterval used by killAwolWorkers to remove unused workers
	KillAwolWorkersInterval int `mapstructure:"killAwolWorkersInterval" toml:"killAwolWorkersInterval" default:"30" commented:"true" comment:"Kill awol worker interval (seconds)" json:"killAwolWorkersInterval"`
}

// HatcheryMicroVM spawns v2 workers in Firecracker microVMs managed through
// the MicroVM custom resource
type HatcheryMicroVM struct {
	hatcheryCommon.Common
	Config     HatcheryConfiguration
	kubeClient kubernetes.Interface
	dynClient  dynamic.Interface
}
