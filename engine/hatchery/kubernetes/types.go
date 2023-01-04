package kubernetes

import (
	"regexp"

	"github.com/ovh/cds/engine/service"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
)

const (
	LABEL_HATCHERY_NAME     = "CDS_HATCHERY_NAME"
	LABEL_WORKER_NAME       = "CDS_WORKER_NAME"
	LABEL_WORKER_MODEL_PATH = "CDS_WORKER_MODEL_PATH"
)

var containerServiceNameRegexp = regexp.MustCompile(`service-([0-9]+)-(.*)`)

// HatcheryConfiguration is the configuration for local hatchery
type HatcheryConfiguration struct {
	service.HatcheryCommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration" json:"commonConfiguration"`
	// DefaultCPU Worker default CPU
	DefaultCPU string `mapstructure:"defaultCPU" toml:"defaultCPU" default:"500m" commented:"false" comment:"Worker default CPU" json:"defaultCPU"`
	// DefaultMemory Worker default memory
	DefaultMemory int `mapstructure:"defaultMemory" toml:"defaultMemory" default:"1024" commented:"false" comment:"Worker default memory in Mo" json:"defaultMemory"`
	// DefaultEphemeralStorage Worker default ephemeral storage size
	DefaultEphemeralStorage string `mapstructure:"defaultEphemeralStorage" toml:"defaultEphemeralStorage" default:"1Gi" commented:"false" comment:"Worker default ephemeral storage size" json:"defaultEphemeralStorage"`
	// DefaultServiceCPU Service default CPU
	DefaultServiceCPU string `mapstructure:"defaultServiceCPU" toml:"defaultServiceCPU" default:"256m" commented:"false" comment:"Service default CPU" json:"defaultServiceCPU"`
	// DefaultServiceMemory Service default memory
	DefaultServiceMemory int `mapstructure:"defaultServiceMemory" toml:"defaultServiceMemory" default:"512" commented:"false" comment:"Service default memory in Mo" json:"defaultServiceMemory"`
	// DefaultServiceEphemeralStorage Service default ephemeral storage size
	DefaultServiceEphemeralStorage string `mapstructure:"defaultServiceEphemeralStorage" toml:"defaultServiceEphemeralStorage" default:"512Mi" commented:"false" comment:"Service default ephemeral storage size" json:"defaultServiceEphemeralStorage"`
	// Namespace is the kubernetes namespace in which workers are spawned"
	Namespace string `mapstructure:"namespace" toml:"namespace" default:"cds" commented:"false" comment:"Kubernetes namespace in which workers are spawned" json:"namespace"`
	// KubernetesMasterURL Address of kubernetes master
	KubernetesMasterURL string `mapstructure:"kubernetesMasterURL" toml:"kubernetesMasterURL" default:"" commented:"false" comment:"Address of kubernetes master" json:"kubernetesMasterURL"`
	// KubernetesConfigFile Kubernetes config file in yaml
	KubernetesConfigFile string `mapstructure:"kubernetesConfigFile" toml:"kubernetesConfigFile" default:"" commented:"false" comment:"Kubernetes config file in yaml" json:"kubernetesConfigFile"`
	// KubernetesUsername Username to connect to kubernetes cluster (optional if config file is set)
	KubernetesUsername string `mapstructure:"username" toml:"username" default:"" commented:"true" comment:"Username to connect to kubernetes cluster (optional if config file is set)" json:"username"`
	// KubernetesPassword Password to connect to kubernetes cluster (optional if config file is set)
	KubernetesPassword string `mapstructure:"password" toml:"password" default:"" commented:"true" comment:"Password to connect to kubernetes cluster (optional if config file is set)" json:"-"`
	// KubernetesToken Token to connect to kubernetes cluster (optional if config file is set)
	KubernetesToken string `mapstructure:"token" toml:"token" default:"" commented:"true" comment:"Token to connect to kubernetes cluster (optional if config file is set)" json:"-"`
	// KubernetesCertAuthData Certificate authority data for tls kubernetes (optional if config file is set)
	KubernetesCertAuthData string `mapstructure:"certAuthorityData" toml:"certAuthorityData" default:"" commented:"true" comment:"Certificate authority data (content, not path and not base64 encoded) for tls kubernetes (optional if no tls needed)" json:"-"`
	// KubernetesClientCertData Client certificate data for tls kubernetes (optional if no tls needed)
	KubernetesClientCertData string `mapstructure:"clientCertData" toml:"clientCertData" default:"" commented:"true" comment:"Client certificate data (content, not path and not base64 encoded) for tls kubernetes (optional if no tls needed)" json:"-"`
	// KubernetesKeyData Client certificate data for tls kubernetes (optional if no tls needed)
	KubernetesClientKeyData string `mapstructure:"clientKeyData" toml:"clientKeyData" default:"" commented:"true" comment:"Client certificate data (content, not path and not base64 encoded) for tls kubernetes (optional if no tls needed)" json:"-"`
	// CustomAnnotations that will be added to pods spawned by the hatchery
	CustomAnnotations []CustomAnnotation `mapstructure:"customAnnotations" toml:"customAnnotations" default:"" commented:"true" comment:"CustomAnnotations that will be added to pods spawned by the hatchery" json:"-"`
}

type CustomAnnotation struct {
	Key   string `mapstructure:"key" toml:"key" default:"" commented:"true" json:"-"`
	Value string `mapstructure:"value" toml:"value" default:"" commented:"true" json:"-"`
}

// HatcheryKubernetes implements HatcheryMode interface for local usage
type HatcheryKubernetes struct {
	hatcheryCommon.Common
	Config     HatcheryConfiguration
	kubeClient KubernetesClient
}
