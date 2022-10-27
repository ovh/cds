package hatchery

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(dbHatchery{}, "hatchery", false, "id"))
	gorpmapping.Register(gorpmapping.New(sdk.HatcheryStatus{}, "hatchery_status", true, "id"))
}

type dbHatchery struct {
	sdk.Hatchery
	gorpmapper.SignedEntity
}

func (o dbHatchery) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{o.ID, o.Name, o.Config}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}{{.Config}}",
	}
}
