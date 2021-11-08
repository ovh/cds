package swarm

import (
	"testing"

	types "github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestHatcherySwarm_getContainers(t *testing.T) {
	h := testSwarmHatchery(t)
	_, err := h.getContainers(context.TODO(), h.dockerClients["default"], types.ContainerListOptions{})
	assert.NoError(t, err)
}

func TestHatcherySwarm_getContainer(t *testing.T) {
	h := testSwarmHatchery(t)
	cs, err := h.getContainers(context.TODO(), h.dockerClients["default"], types.ContainerListOptions{})
	assert.NoError(t, err)
	if len(cs) > 0 {
		c, err := getContainer(h.dockerClients["default"], cs, cs[0].Names[0], types.ContainerListOptions{})
		assert.NoError(t, err)
		t.Logf("container found: %+v", c)
	}
}
