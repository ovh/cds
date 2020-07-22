package ascode

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

type dbAsCodeEvents sdk.AsCodeEvent

func init() {
	gorpmapping.Register(gorpmapping.New(dbAsCodeEvents{}, "as_code_events", true, "id"))
}
