package user

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type authentifiedUser struct {
	sdk.AuthentifiedUser
	gorpmapper.SignedEntity
}

func (u authentifiedUser) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Username}}{{.Fullname}}{{.Ring}}{{printDate .Created}}",
	}
}

type userContact struct {
	sdk.UserContact
	gorpmapper.SignedEntity
}

func (c userContact) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.UserID}}{{.Type}}{{.Value}}{{.Primary}}{{.Verified}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(authentifiedUser{}, "authentified_user", false, "id"))
	gorpmapping.Register(gorpmapping.New(userContact{}, "user_contact", true, "id"))
}
