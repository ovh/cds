package local

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type userRegistration struct {
	sdk.UserRegistration
	gorpmapper.SignedEntity
}

func (u userRegistration) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{u.ID, u.Username, u.Fullname, u.Email, u.Hash} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Username}}{{.Fullname}}{{.Email}}{{.Hash}}",
	}
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(userRegistration{}, "user_registration", false, "id"),
	)
}
