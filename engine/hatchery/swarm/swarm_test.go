package swarm

import (
	"testing"

	docker "github.com/docker/docker/client"
	"github.com/ovh/cds/sdk/log"
)

func init() {
	log.Initialize(&log.Conf{Level: "debug"})
}

func testSwarmHatchery(t *testing.T) *HatcherySwarm {
	log.SetLogger(t)
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
