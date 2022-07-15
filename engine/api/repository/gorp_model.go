package repository

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(gorpmapping.New(dbProjectRepository{}, "project_repository", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbProjectRepositoryAnalyze{}, "project_repository_analyze", false, "id"))
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

type dbProjectRepositoryAnalyze struct {
	sdk.ProjectRepositoryAnalyze
	gorpmapper.SignedEntity
}

func (v dbProjectRepositoryAnalyze) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{v.ID, v.ProjectRepositoryID, v.VCSProjectID, v.ProjectKey, v.Commit}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.ProjectRepositoryID}}{{.VCSProjectID}}{{.ProjectKey}}{{.Commit}}",
	}
}
