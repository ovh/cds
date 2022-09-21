package organization

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(dbOrganization{}, "organization", false, "id"))
}

type dbOrganization struct {
	sdk.Organization
	gorpmapper.SignedEntity
}

func (o dbOrganization) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{o.ID, o.Name}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}",
	}
}
