package scheduler

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

func TestSchedulerRun(t *testing.T) {
	api, db, router := newTestAPI(t)
	exs, status, err := Run(db)
	if err != nil {
		t.Fatalf("%s: %s", status, err)
	}
	t.Logf("Has prepare %v", exs)
}
