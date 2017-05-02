package workflow

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
)

func TestLoadAll(t *testing.T) {
	db := test.SetupPG(t)

	key := test.RandomString(t, 10)
	proj := assets.InsertTestProject(t, db, key, key, nil)

	ws, err := LoadAll(db, proj.Key)

}
