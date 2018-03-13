package swarm

import (
	"testing"

	docker "github.com/docker/docker/client"
)

func testSwarmHatchery(t *testing.T) *HatcherySwarm {
	dockerClient, err := docker.NewEnvClient()
	if err != nil {
		t.Logf("unable to get docker client: %v. Skipping this test", err)
		t.SkipNow()
	}

	h := &HatcherySwarm{
		dockerClient: dockerClient,
	}
	return h
}
