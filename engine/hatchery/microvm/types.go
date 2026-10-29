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

	// platformBootArgs is the platform default kernel command line, which
	// spec.bootArgs replaces
	platformBootArgs = "console=ttyS0 reboot=k panic=1 pci=off"
	// defaultExtraBootArgs keeps the serial console quiet (it is slow in
	// Firecracker), skips the probe of the PS/2 controller the VM does not
	// have and the clock checks a VM does not need, seeds the kernel random
	// generator from the CPU instead of waiting for entropy (about 1 s of
	// kernel boot in all), and masks the units of the image a CI worker does
	// not need
	defaultExtraBootArgs = "quiet i8042.noaux i8042.nomux i8042.nopnp i8042.dumbkbd tsc=reliable no_timer_check random.trust_cpu=on systemd.mask=serial-getty@ttyS0.service systemd.mask=getty@tty1.service systemd.mask=apt-daily.timer systemd.mask=apt-daily-upgrade.timer systemd.mask=unattended-upgrades.service systemd.mask=motd-news.timer systemd.mask=man-db.timer systemd.mask=logrotate.timer systemd.mask=e2scrub_all.timer systemd.mask=fstrim.timer"
	// sshBootArgs masks the SSH server when debug SSH is off
	sshBootArgs = "systemd.mask=ssh.service systemd.mask=ssh.socket"
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
	// ExtraBootArgs appended to the platform default kernel command line
	ExtraBootArgs string `mapstructure:"extraBootArgs" toml:"extraBootArgs" default:"quiet i8042.noaux i8042.nomux i8042.nopnp i8042.dumbkbd tsc=reliable no_timer_check random.trust_cpu=on systemd.mask=serial-getty@ttyS0.service systemd.mask=getty@tty1.service systemd.mask=apt-daily.timer systemd.mask=apt-daily-upgrade.timer systemd.mask=unattended-upgrades.service systemd.mask=motd-news.timer systemd.mask=man-db.timer systemd.mask=logrotate.timer systemd.mask=e2scrub_all.timer systemd.mask=fstrim.timer" commented:"true" comment:"Kernel arguments appended to the platform default command line of the workers (systemd.mask=<unit> skips a unit of the image). The SSH server is masked too unless debugSSHAuthorizedKeys is set" json:"extraBootArgs"`
	// PreloadedPlugins are written in the worker basedir from the platform cache
	PreloadedPlugins []string `mapstructure:"preloadedPlugins" toml:"preloadedPlugins" default:"" commented:"true" comment:"Plugins whose binary the platform caches on its nodes and writes in the worker basedir, so that the worker does not download them at each job (script runs the run steps)" json:"preloadedPlugins"`
	// Flavors are the worker sizes jobs and worker models choose from
	Flavors []FlavorConfig `mapstructure:"flavors" toml:"flavors" commented:"true" comment:"Worker sizes. A job chooses one with runs-on.flavor, a worker model with spec.flavor. Defaults to XS (1 vCPU, 4096 MiB, 10 GiB), S (2, 8192, 20), M (4, 16384, 50) and L (6, 24576, 100)" json:"flavors"`
	// DefaultFlavor used when neither the job nor the worker model sets one
	DefaultFlavor string `mapstructure:"defaultFlavor" toml:"defaultFlavor" default:"XS" commented:"true" comment:"Flavor of the workers whose job and worker model set none" json:"defaultFlavor"`
	// RegistrationTimeout delay after which a booted MicroVM without a registered worker is deleted
	RegistrationTimeout int `mapstructure:"registrationTimeout" toml:"registrationTimeout" default:"600" commented:"true" comment:"Delete a MicroVM whose worker is not registered after this delay (seconds), counted from the boot, or from the creation while not booted" json:"registrationTimeout"`
	// QuotaPendingTimeout delay after which a MicroVM refused by the namespace quota is deleted
	QuotaPendingTimeout int `mapstructure:"quotaPendingTimeout" toml:"quotaPendingTimeout" default:"1800" commented:"true" comment:"Delete a MicroVM still refused by the namespace quota after this delay (seconds). No new MicroVM is created while one is refused" json:"quotaPendingTimeout"`
	// KillAwolWorkersInterval used by killAwolWorkers to remove unused workers
	KillAwolWorkersInterval int `mapstructure:"killAwolWorkersInterval" toml:"killAwolWorkersInterval" default:"30" commented:"true" comment:"Kill awol worker interval (seconds)" json:"killAwolWorkersInterval"`
}

// FlavorConfig is a worker size
type FlavorConfig struct {
	Name      string `mapstructure:"name" toml:"name" json:"name"`
	VCPUs     int    `mapstructure:"vcpus" toml:"vcpus" json:"vcpus"`
	MemoryMiB int    `mapstructure:"memoryMiB" toml:"memoryMiB" json:"memoryMiB"`
	// DiskGiB is raised to the rootfs size of the worker model when smaller
	DiskGiB int `mapstructure:"diskGiB" toml:"diskGiB" json:"diskGiB"`
}

// defaultFlavors apply when the configuration sets none
var defaultFlavors = []FlavorConfig{
	{Name: "XS", VCPUs: 1, MemoryMiB: 4096, DiskGiB: 10},
	{Name: "S", VCPUs: 2, MemoryMiB: 8192, DiskGiB: 20},
	{Name: "M", VCPUs: 4, MemoryMiB: 16384, DiskGiB: 50},
	{Name: "L", VCPUs: 6, MemoryMiB: 24576, DiskGiB: 100},
}

// HatcheryMicroVM spawns v2 workers in Firecracker microVMs managed through
// the MicroVM custom resource
type HatcheryMicroVM struct {
	hatcheryCommon.Common
	Config     HatcheryConfiguration
	kubeClient kubernetes.Interface
	dynClient  dynamic.Interface

	workerBinaries workerBinaryCache
	pluginBinaries pluginBinaryCache
}
