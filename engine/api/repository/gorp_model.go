package repository

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(dbProjectRepository{}, "project_repository", false, "id"))
}

type dbProjectRepository struct {
	sdk.ProjectRepository
	gorpmapper.SignedEntity
}

func (v dbProjectRepository) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{v.ID, v.Name, v.VCSProjectID}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}{{.VCSProjectID}}",
	}
}
