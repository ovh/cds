package entity

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(dbEntity{}, "entity", false, "id"))
}

type dbEntity struct {
	sdk.Entity
	gorpmapper.SignedEntity
}

func (v dbEntity) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{v.ID, v.Name, v.ProjectKey, v.ProjectRepositoryID, v.Type, v.Ref, v.Commit, v.Data}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}{{.ProjectKey}}{{.ProjectRepositoryID}}{{.Type}}{{.Ref}}{{.Commit}}{{hash .Data}}",
	}
}
