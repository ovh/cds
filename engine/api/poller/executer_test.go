package poller

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

func TestExecuterRun(t *testing.T) {
	api, db, router := newTestAPI(t)
	exs, err := ExecuterRun(db)
	if err != nil {
		t.Fatalf("%s", err)
	}
	t.Logf("Has execute %v", exs)
}
