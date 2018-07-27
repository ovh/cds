package swarm

import (
	"sync"
	"testing"

	docker "github.com/docker/docker/client"
	"github.com/ovh/cds/sdk/log"
	context "golang.org/x/net/context"
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

	h := &HatcherySwarm{
		dockerClients: map[string]*dockerClient{},
	}
	h.dockerClients["default"] = &dockerClient{Client: *c, MaxContainers: 2, name: "default", pullImageMutex: &sync.Mutex{}}
	return h
}
