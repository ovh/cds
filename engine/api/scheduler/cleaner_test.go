package scheduler

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

func TestCleanerRun(t *testing.T) {
	test.SetupPG(t)

	exs, err := CleanerRun(2)
	test.NoError(t, err)
	t.Logf("Has deleted %v", exs)
}
