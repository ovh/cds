package storage

import (
	"time"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InitDBMapping(m *gorpmapper.Mapper) {
	m.Register(m.NewTableMapping(Unit{}, "storage_unit", false, "id"))
	m.Register(m.NewTableMapping(ItemUnit{}, "storage_unit_index", false, "id"))
}

type Unit struct {
	gorpmapper.SignedEntity
	ID      string            `json:"id" db:"id"`
	Created time.Time         `json:"created" db:"created"`
	Name    string            `json:"name" db:"name"`
	Config  sdk.ServiceConfig `json:"config" db:"config"`
}

func (i Unit) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{i.ID, i.Name} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}",
	}
}

type ItemUnit struct {
	gorpmapper.SignedEntity
	ID           string    `json:"id" db:"id"`
	ItemID       string    `json:"item_id" db:"item_id"`
	UnitID       string    `json:"unit_id" db:"unit_id"`
	Created      time.Time `json:"created" db:"created"`
	LastModified time.Time `json:"last_modified" db:"last_modified"`
	Complete     bool      `json:"complete" db:"complete"`
	Locator      string    `json:"-" db:"cipher_locator" gorpmapping:"encrypted,UnitID,ItemID"`
}

func (i ItemUnit) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{i.ID, i.ItemID, i.UnitID} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.ItemID}}{{.UnitID}}",
	}
}
