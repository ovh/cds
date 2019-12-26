package swarm

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	context "golang.org/x/net/context"
)

func TestHatcherySwarm_killAwolNetworks(t *testing.T) {
	h := testSwarmHatchery(t)
	err := h.killAwolNetworks(context.Background())
	test.NoError(t, err)
}
