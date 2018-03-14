package swarm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHatcherySwarm_getContainers(t *testing.T) {
	h := testSwarmHatchery(t)
	_, err := h.getContainers()
	assert.NoError(t, err)
}

func TestHatcherySwarm_getContainer(t *testing.T) {
	h := testSwarmHatchery(t)
	cs, err := h.getContainers()
	assert.NoError(t, err)
	if len(cs) > 0 {
		c, err := h.getContainer(cs[0].Names[0])
		assert.NoError(t, err)
		t.Logf("container found: %+v", c)
	}
}
