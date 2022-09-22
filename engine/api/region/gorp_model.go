package region

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(dbRegion{}, "region", false, "id"))
}

type dbRegion struct {
	sdk.Region
	gorpmapper.SignedEntity
}

func (o dbRegion) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{o.ID, o.Name}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}",
	}
}
