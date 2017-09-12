package scheduler

import (
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
)

func TestSchedulerRun(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)
	exs, status, err := Run(db)
	if err != nil {
		t.Fatalf("%s: %s", status, err)
	}
	t.Logf("Has prepare %v", exs)
}
