package scheduler

import (
	"testing"

	_ "github.com/ovh/cds/engine/api/test"
)

func TestSchedulerRun(t *testing.T) {
	exs, status, err := Run()
	if err != nil {
		t.Fatalf("%s: %s", status, err)
	}
	t.Logf("Has prepare %v", exs)
}
