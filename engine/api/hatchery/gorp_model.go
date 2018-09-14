package hatchery

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// Hatchery is a gorp wrapper around sdk.Hatchery
type Hatchery sdk.Hatchery

func init() {
	gorpmapping.Register(gorpmapping.New(Hatchery{}, "hatchery", true, "id"))
}
