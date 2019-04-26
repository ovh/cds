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
	hatchery.CommonConfiguration `mapstructure:"commonConfiguration" toml:"commonConfiguration" json:"commonConfiguration"`
	NbProvision                  int    `mapstructure:"nbProvision" toml:"nbProvision" default:"1" comment:"Nb Workers to provision" json:"nbProvision"`
	WorkerTTL                    int    `mapstructure:"workerTTL" toml:"workerTTL" default:"10" commented:"false" comment:"Worker TTL (minutes)" json:"workerTTL"`
	DefaultMemory                int    `mapstructure:"defaultMemory" toml:"defaultMemory" default:"1024" commented:"false" comment:"Worker default memory in Mo" json:"defaultMemory"`
	Namespace                    string `mapstructure:"namespace" toml:"namespace" default:"cds" commented:"false" comment:"Kubernetes namespace in which workers are spawned" json:"namespace"`
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
