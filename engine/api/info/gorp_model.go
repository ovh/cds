package info

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

// Info is a gorp wrapper around sdk.Info
type Info sdk.Info

func init() {
	gorpmapping.Register(gorpmapping.New(Info{}, "info", true, "id"))
}
