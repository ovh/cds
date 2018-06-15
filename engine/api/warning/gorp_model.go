package warning

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type warning sdk.Warning

func init() {
	gorpmapping.Register(
		gorpmapping.New(warning{}, "warning", true, "id"),
	)
}
