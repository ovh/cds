package item

import (
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InitDBMapping(m *gorpmapper.Mapper) {
	m.Register(m.NewTableMapping(cdnItemDB{}, "item", false, "id"))
}

type cdnItemDB struct {
	gorpmapper.SignedEntity
	sdk.CDNItem
}

func toItemDB(c sdk.CDNItem) *cdnItemDB {
	return &cdnItemDB{CDNItem: c}
}

func (i cdnItemDB) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{i.ID, i.APIRefHash, i.Type} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.APIRefHash}}{{.Type}}",
	}
}
