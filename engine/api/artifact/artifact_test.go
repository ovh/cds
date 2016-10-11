package artifact

import (
	"testing"

	"github.com/proullon/ramsql/engine/log"
	"github.com/ovh/cds/engine/api/test"
)

func TestCreateBuiltinArtifactActions(t *testing.T) {
	log.UseTestLogger(t)
	db := test.Setup("TestCreateBuiltinArtifactActions", t)

	err := CreateBuiltinArtifactActions(db)
	if err != nil {
		t.Fatalf("Cannot create builtin artifact actions: %s", err)
	}
}
