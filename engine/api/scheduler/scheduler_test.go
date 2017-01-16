package scheduler

import (
	"testing"

	_ "github.com/ovh/cds/engine/api/test"
)

func TestSchedulerRun(t *testing.T) {
	exs, err := SchedulerRun()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Has prepare %v", exs)
}
