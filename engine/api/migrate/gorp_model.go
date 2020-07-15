package migrate

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

func init() {
	gorpmapping.Register(
		gorpmapping.New(sdk.Migration{}, "cds_migration", true, "id"),
	)
}
