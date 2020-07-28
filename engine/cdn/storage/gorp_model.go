package storage

import (
	"time"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func initDBMapping(m *gorpmapper.Mapper) {
	m.Register(m.NewTableMapping(Unit{}, "storage_unit", false, "id"))
}

type UnitConfig interface{}

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
