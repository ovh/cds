package storage

import (
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InitDBMapping(m *gorpmapper.Mapper) {
	m.Register(m.NewTableMapping(unitDB{}, "storage_unit", false, "id"))
	m.Register(m.NewTableMapping(itemUnitDB{}, "storage_unit_item", false, "id"))
}

type itemUnitDB struct {
	gorpmapper.SignedEntity
	sdk.CDNItemUnit
}

func toItemUnitDB(c sdk.CDNItemUnit) *itemUnitDB {
	return &itemUnitDB{CDNItemUnit: c}
}

type unitDB struct {
	gorpmapper.SignedEntity
	sdk.CDNUnit
}

func toUnitDB(c sdk.CDNUnit) *unitDB {
	return &unitDB{CDNUnit: c}
}

func (i itemUnitDB) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{i.ID, i.ItemID, i.UnitID} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.ItemID}}{{.UnitID}}",
	}
}

func (i unitDB) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{i.ID, i.Name} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}",
	}
}
