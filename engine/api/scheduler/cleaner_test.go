package scheduler

import (
	"testing"

	"github.com/ovh/cds/engine/api/testwithdb"
)

func TestCleanerRun(t *testing.T) {
	if testwithdb.DBDriver == "" {
		t.SkipNow()
		return
	}
	testwithdb.SetupPG(t)

	exs, err := CleanerRun(0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Has deleted %v", exs)

}
