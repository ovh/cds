package ascode

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type dbAsCodeEvents sdk.AsCodeEvent

func init() {
	gorpmapping.Register(gorpmapping.New(dbAsCodeEvents{}, "as_code_events", true, "id"))
}
