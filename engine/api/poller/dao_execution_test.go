package poller

import (
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
)

func TestLoadPendingExecutions(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	pe, err := LoadPendingExecutions(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", pe)
}
