package marathon

import (
	"time"

	"github.com/gambol99/go-marathon"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/sdk/cdsclient"
)

type marathonJDD struct {
	MaxProvision       int
	MaxWorker          int
	DeploymentTime     int
	WorkerSpawnTimeout int
	Prefix             string
}

func InitMarathonMarathonTest(opts marathonJDD) *HatcheryMarathon {
	h := New()
	config := marathon.NewDefaultConfig()
	config.URL = "http://mara.thon"
	config.HTTPBasicAuthUser = "foo"
	config.HTTPBasicPassword = "bar"
	config.HTTPClient = cdsclient.NewHTTPClient(time.Minute, false)

	gock.InterceptClient(config.HTTPClient)

	h.marathonClient, _ = marathon.NewClient(config)
	h.Config.Provision.MaxConcurrentProvisioning = opts.MaxProvision
	h.Config.Provision.MaxWorker = opts.MaxWorker
	h.Config.DefaultCPUs = 1
	if opts.Prefix != "" {
		h.Config.MarathonIDPrefix = opts.Prefix
	}

	if opts.WorkerSpawnTimeout > 0 {
		h.Config.WorkerSpawnTimeout = opts.WorkerSpawnTimeout
	}
	h.Client = cdsclient.New(cdsclient.Config{Host: "http://cds-api.local", InsecureSkipVerifyTLS: false})
	gock.InterceptClient(h.Client.(cdsclient.Raw).HTTPClient())
	return h
}
