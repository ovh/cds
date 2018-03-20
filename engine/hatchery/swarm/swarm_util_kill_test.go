package swarm

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

func TestHatcherySwarm_killAwolNetworks(t *testing.T) {
	h := testSwarmHatchery(t)
	err := h.killAwolNetworks()
	test.NoError(t, err)
}
