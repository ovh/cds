package featureflipping

import (
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func Init(m *gorpmapper.Mapper) {
	m.Register(m.NewTableMapping(sdk.Feature{}, "feature_flipping", true, "id"))
}
