package kubernetes

import (
	"os/exec"
	"sync"
	"time"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"

	"k8s.io/client-go/kubernetes"
)

const (
	LABEL_HATCHERY_NAME = "CDS_HATCHERY_NAME"
	LABEL_WORKER        = "CDS_WORKER"
	LABEL_WORKER_MODEL  = "CDS_WORKER_MODEL"
)

// HatcheryConfiguration is the configuration for local hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration"`
	NbProvision                  int `mapstructure:"nbProvision" toml:"nbProvision" default:"1" comment:"Nb Workers to provision"`
	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `mapstructure:"workerTTL" toml:"workerTTL" default:"10" commented:"false" comment:"Worker TTL (minutes)"`
	// DefaultMemory Worker default memory
	DefaultMemory int `mapstructure:"defaultMemory" toml:"defaultMemory" default:"1024" commented:"false" comment:"Worker default memory in Mo"`
	// KubernetesNamespace is the kubernetes namespace in which workers are spawned"
	KubernetesNamespace string `mapstructure:"namespace" toml:"namespace" default:"default" commented:"false" comment:"Kubernetes namespace in which workers are spawned"`
	// KubernetesMasterURL Address of kubernetes master
	KubernetesMasterURL string `mapstructure:"kubernetesMasterURL" toml:"kubernetesMasterURL" default:"https://1.1.1.1:8443" commented:"false" comment:"Address of kubernetes master"`
	// KubernetesConfigFile Kubernetes config file in yaml
	KubernetesConfigFile string `mapstructure:"kubernetesConfigFile" toml:"kubernetesConfigFile" default:"kubeconfig.yaml" commented:"false" comment:"Kubernetes config file in yaml"`
	// KubernetesUsername Username to connect to kubernetes cluster (optional if config file is set)
	KubernetesUsername string `mapstructure:"username" toml:"username" default:"" commented:"true" comment:"Username to connect to kubernetes cluster (optional if config file is set)"`
	// KubernetesPassword Password to connect to kubernetes cluster (optional if config file is set)
	KubernetesPassword string `mapstructure:"password" toml:"password" default:"" commented:"true" comment:"Password to connect to kubernetes cluster (optional if config file is set)"`
	// KubernetesToken Token to connect to kubernetes cluster (optional if config file is set)
	KubernetesToken string `mapstructure:"token" toml:"token" default:"" commented:"true" comment:"Token to connect to kubernetes cluster (optional if config file is set)"`
	// KubernetesCertAuthData Certificate authority data for tls kubernetes (optional if config file is set)
	KubernetesCertAuthData string `mapstructure:"certAuthorityData" toml:"certAuthorityData" default:"" commented:"true" comment:"Certificate authority data (content, not path and not base64 encoded) for tls kubernetes (optional if no tls needed)"`
	// KubernetesClientCertData Client certificate data for tls kubernetes (optional if no tls needed)
	KubernetesClientCertData string `mapstructure:"clientCertData" toml:"clientCertData" default:"" commented:"true" comment:"Client certificate data (content, not path and not base64 encoded) for tls kubernetes (optional if no tls needed)"`
	// KubernetesKeyData Client certificate data for tls kubernetes (optional if no tls needed)
	KubernetesClientKeyData string `mapstructure:"clientKeyData" toml:"clientKeyData" default:"" commented:"true" comment:"Client certificate data (content, not path and not base64 encoded) for tls kubernetes (optional if no tls needed)"`
}

// HatcheryKubernetes implements HatcheryMode interface for local usage
type HatcheryKubernetes struct {
	hatcheryCommon.Common
	Config HatcheryConfiguration
	sync.Mutex
	hatch     *sdk.Hatchery
	workers   map[string]workerCmd
	client    cdsclient.Interface
	os        string
	arch      string
	k8sClient *kubernetes.Clientset
}

type workerCmd struct {
	cmd     *exec.Cmd
	created time.Time
}
