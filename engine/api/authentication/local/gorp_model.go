package local

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

type userRegistration struct {
	sdk.UserRegistration
	gorpmapping.SignedEntity
}

func (u userRegistration) Canonical() gorpmapping.CanonicalForms {
	_ = []interface{}{u.ID, u.Username, u.Fullname, u.Email, u.Hash} // Checks that fields exists at compilation
	return []gorpmapping.CanonicalForm{
		"{{.ID}}{{.Username}}{{.Fullname}}{{.Email}}{{.Hash}}",
	}
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(userRegistration{}, "user_registration", false, "id"),
	)
}
