package swarm

import (
	"testing"
	"time"

	docker "github.com/moby/moby/client"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	cdslog "github.com/ovh/cds/sdk/log"
)

func init() {
	cdslog.Initialize(context.TODO(), &cdslog.Conf{Level: "debug"})
}

func testSwarmHatchery(t *testing.T) *HatcherySwarm {
	log.Factory = log.NewTestingWrapper(t)
	c, err := docker.NewClientWithOpts(docker.FromEnv)
	if err != nil {
		t.Skipf("unable to get docker client: %v. Skipping this test", err)
		return nil
	}

	if _, err := c.Info(context.Background()); err != nil {
		t.Skipf("unable to ping docker client: %v. Skipping this test", err)
		return nil
	}

	httpClient := cdsclient.NewHTTPClient(1*time.Minute, false)
	docker.WithHTTPClient(httpClient)

	h := &HatcherySwarm{
		dockerClients: map[string]*dockerClient{},
		Config:        HatcheryConfiguration{},
		Common:        hatchery.Common{},
	}
	h.dockerClients["default"] = &dockerClient{Client: *c, MaxContainers: 2, name: "default"}

	gock.InterceptClient(httpClient)
	return h
}

func InitTestHatcherySwarm(t *testing.T) *HatcherySwarm {
	httpClient := cdsclient.NewHTTPClient(1*time.Minute, false)
	c, err := docker.NewClientWithOpts(
		docker.WithHTTPClient(httpClient),
		docker.WithHost("https://lolcat.local"),
		docker.WithVersion("6.66"),
	)
	require.NoError(t, err)

	gock.InterceptClient(httpClient)

	h := &HatcherySwarm{
		dockerClients: map[string]*dockerClient{},
		Config:        HatcheryConfiguration{},
	}
	h.ServiceInstance = &sdk.Service{
		CanonicalService: sdk.CanonicalService{
			ID:   1,
			Name: "my-hatchery",
		},
	}
	h.dockerClients["default"] = &dockerClient{Client: *c, name: "default"}

	h.Client = cdsclient.New(cdsclient.Config{Host: "https://cds-api.local"})
	gock.InterceptClient(h.Client.HTTPClient())
	return h
}
