package featureflipping

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
)

func init() {
	gorpmapping.Register(
		gorpmapping.New(sdk.Feature{}, "feature_flipping", true, "id"),
	)
}
