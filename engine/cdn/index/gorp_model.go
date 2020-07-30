package index

import (
	"time"

	"github.com/ovh/cds/engine/gorpmapper"
)

func InitDBMapping(m *gorpmapper.Mapper) {
	m.Register(m.NewTableMapping(Item{}, "index", false, "id"))
}

type Item struct {
	gorpmapper.SignedEntity
	ID      string    `json:"id" db:"id"`
	Created time.Time `json:"created" db:"created"`
	Name    string    `json:"name" db:"name"`
}

func (i Item) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{i.ID, i.Name} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}",
	}
}
