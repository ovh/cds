package index

import (
	"github.com/ovh/cds/engine/gorpmapper"
)

func InitDBMapping(m *gorpmapper.Mapper) {
	m.Register(m.NewTableMapping(Item{}, "index", false, "id"))
}

func (i Item) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{i.ID, i.ApiRefHash, i.Type} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.ApiRefHash}}{{.Type}}",
	}
}
