package organization

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(dbOrganization{}, "organization", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbOrganizationRegion{}, "organization_region", false, "id"))
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

type dbOrganizationRegion struct {
	sdk.OrganizationRegion
	gorpmapper.SignedEntity
}

func (o dbOrganizationRegion) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{o.ID, o.OrganizationID, o.RegionID}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.OrganizationID}}{{.RegionID}}",
	}
}
