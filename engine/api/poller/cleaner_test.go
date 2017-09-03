package poller

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

func TestCleanerRun(t *testing.T) {
	api, db, router := newTestAPI(t)

	exs, err := CleanerRun(db, 2)
	test.NoError(t, err)
	t.Logf("Has deleted %v", exs)
}
