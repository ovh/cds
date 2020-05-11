package swarm

import (
	"github.com/ovh/cds/engine/hatchery"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
	"testing"
	"time"

	docker "github.com/docker/docker/client"
	"github.com/ovh/cds/sdk/log"
	"golang.org/x/net/context"
)

func init() {
	log.Initialize(&log.Conf{Level: "debug"})
}

func testSwarmHatchery(t *testing.T) *HatcherySwarm {
	log.SetLogger(t)
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
		Config: HatcheryConfiguration{
			DisableDockerOptsOnRequirements: false,
		},
		Common: hatchery.Common{},
	}
	h.dockerClients["default"] = &dockerClient{Client: *c, MaxContainers: 2, name: "default"}

	gock.InterceptClient(httpClient)
	return h
}

func InitTestHatcherySwarm(t *testing.T) *HatcherySwarm {
	httpClient := cdsclient.NewHTTPClient(1*time.Minute, false)
	c, err := docker.NewClientWithOpts(
		docker.WithHTTPClient(httpClient),
		docker.WithHost("https://lolcat.host"),
		docker.WithVersion("6.66"),
	)
	require.NoError(t, err)

	gock.InterceptClient(httpClient)

	h := &HatcherySwarm{
		dockerClients: map[string]*dockerClient{},
		Config: HatcheryConfiguration{
			DisableDockerOptsOnRequirements: false,
		},
	}
	h.dockerClients["default"] = &dockerClient{Client: *c, name: "default"}

	h.Client = cdsclient.New(cdsclient.Config{Host: "https://lolcat.api"})
	gock.InterceptClient(h.Client.HTTPClient())
	return h
}
