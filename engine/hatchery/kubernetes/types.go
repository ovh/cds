package kubernetes

import (
	"os/exec"
	"regexp"
	"sync"
	"time"

	"github.com/ovh/cds/engine/service"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/sdk/cdsclient"

	"k8s.io/client-go/kubernetes"
)

const (
	LABEL_HATCHERY_NAME = "CDS_HATCHERY_NAME"
	LABEL_WORKER        = "CDS_WORKER"
	LABEL_SECRET        = "CDS_SECRET"
	LABEL_WORKER_MODEL  = "CDS_WORKER_MODEL"
)

var containerServiceNameRegexp = regexp.MustCompile(`service-([0-9]+)-(.*)`)

// HatcheryConfiguration is the configuration for local hatchery
type HatcheryConfiguration struct {
	service.HatcheryCommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration" json:"commonConfiguration"`
	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `mapstructure:"workerTTL" toml:"workerTTL" default:"10" commented:"false" comment:"Worker TTL (minutes)" json:"workerTTL"`
	// DefaultMemory Worker default memory
	DefaultMemory int `mapstructure:"defaultMemory" toml:"defaultMemory" default:"1024" commented:"false" comment:"Worker default memory in Mo" json:"defaultMemory"`
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
}

// HatcheryKubernetes implements HatcheryMode interface for local usage
type HatcheryKubernetes struct {
	hatcheryCommon.Common
	Config HatcheryConfiguration
	sync.Mutex
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
