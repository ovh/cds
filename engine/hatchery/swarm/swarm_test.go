package swarm

import (
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
	dockerClient, err := docker.NewEnvClient()
	if err != nil {
		t.Skipf("unable to get docker client: %v. Skipping this test", err)
		return nil
	}

	if _, err := dockerClient.Info(context.Background()); err != nil {
		t.Skipf("unable to ping docker client: %v. Skipping this test", err)
		return nil
	}

	h := &HatcherySwarm{
		dockerClient: dockerClient,
	}
	return h
}
