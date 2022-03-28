package vcs

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(dbVCSProject{}, "vcs_project", false, "id"))
}

type dbVCSProject struct {
	sdk.VCSProject
	gorpmapper.SignedEntity
}

func (v dbVCSProject) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{v.ID, v.Name, v.ProjectID, v.Type}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}{{.ProjectID}}{{.Type}}",
	}
}
