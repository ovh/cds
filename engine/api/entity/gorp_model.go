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

// InitiatorCanonical is the owner identity bound into the signature. A stored owner always yields a
// value ("||" when nobody is identified) so it cannot be confused with a NULL, not yet migrated, column.
func (v dbEntity) InitiatorCanonical() string {
	if v.Initiator == nil {
		return ""
	}
	return v.Initiator.UserID + "|" + v.Initiator.VCS + "|" + v.Initiator.VCSUsername
}

func (v dbEntity) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{v.ID, v.Name, v.ProjectKey, v.ProjectRepositoryID, v.Type, v.Ref, v.Commit, v.Data, v.InitiatorCanonical()}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}{{.ProjectKey}}{{.ProjectRepositoryID}}{{.Type}}{{.Ref}}{{.Commit}}{{md5sum .Data}}{{.InitiatorCanonical}}",
		"{{.ID}}{{.Name}}{{.ProjectKey}}{{.ProjectRepositoryID}}{{.Type}}{{.Ref}}{{.Commit}}{{md5sum .Data}}",
		"{{.ID}}{{.Name}}{{.ProjectKey}}{{.ProjectRepositoryID}}{{.Type}}{{.Ref}}{{.Commit}}{{hash .Data}}",
	}
}
