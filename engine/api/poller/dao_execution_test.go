package poller

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
)

func TestLoadPendingExecutions(t *testing.T) {
	db, _ := test.SetupPG(t)
	pe, err := LoadPendingExecutions(db)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", pe)
}
