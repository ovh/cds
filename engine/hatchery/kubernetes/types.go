package kubernetes

import (
	"os/exec"
	"regexp"
	"sync"
	"time"

	hatcheryCommon "github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/hatchery"

	"k8s.io/client-go/kubernetes"
)

const (
	LABEL_HATCHERY_NAME  = "CDS_HATCHERY_NAME"
	LABEL_WORKER         = "CDS_WORKER"
	LABEL_WORKER_MODEL   = "CDS_WORKER_MODEL"
	LABEL_SERVICE_JOB_ID = "CDS_SERVICE_JOB_ID"
)

var containerServiceNameRegexp = regexp.MustCompile(`service-([0-9]+)-(.*)`)

// HatcheryConfiguration is the configuration for local hatchery
type HatcheryConfiguration struct {
	hatchery.CommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration"`
	NbProvision                  int `mapstructure:"nbProvision" toml:"nbProvision" default:"1" comment:"Nb Workers to provision"`
	// WorkerTTL Worker TTL (minutes)
	WorkerTTL int `mapstructure:"workerTTL" toml:"workerTTL" default:"10" commented:"false" comment:"Worker TTL (minutes)"`
	// DefaultMemory Worker default memory
	DefaultMemory int `mapstructure:"defaultMemory" toml:"defaultMemory" default:"1024" commented:"false" comment:"Worker default memory in Mo"`
	// KubernetesMasterURL Worker default memory
	KubernetesNamespace string `mapstructure:"namespace" toml:"namespace" default:"default" commented:"false" comment:"Kubernetes namespace in which workers are spawned"`
	// KubernetesMasterURL Worker default memory
	KubernetesMasterURL string `mapstructure:"kubernetesMasterURL" toml:"kubernetesMasterURL" default:"https://1.1.1.1:8443" commented:"false" comment:"Address of kubernetes master"`
	// KubernetesUsername Worker default memory
	KubernetesUsername string `mapstructure:"username" toml:"username" default:"" commented:"true" comment:"Username to connect to kubernetes cluster"`
	// KubernetesPassword Worker default memory
	KubernetesPassword string `mapstructure:"password" toml:"password" default:"" commented:"true" comment:"Password to connect to kubernetes cluster"`
	// KubernetesToken Worker default memory
	KubernetesToken string `mapstructure:"token" toml:"token" default:"" commented:"true" comment:"Token to connect to kubernetes cluster"`
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
