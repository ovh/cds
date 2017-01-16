package scheduler

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

func TestSchedulerRun(t *testing.T) {
	test.SetupPG(t)
	exs, err := SchedulerRun()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Has prepare %v", exs)
}
