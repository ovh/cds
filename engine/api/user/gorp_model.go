package user

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

type authentifiedUser struct {
	sdk.AuthentifiedUser
	gorpmapping.SignedEntity
}

func (u authentifiedUser) Canonical() gorpmapping.CanonicalForms {
	return []gorpmapping.CanonicalForm{
		"{{.ID}}{{.Username}}{{.Fullname}}{{.Ring}}{{printDate .Created}}",
	}
}

type userContact struct {
	sdk.UserContact
	gorpmapping.SignedEntity
}

func (c userContact) Canonical() gorpmapping.CanonicalForms {
	return []gorpmapping.CanonicalForm{
		"{{.ID}}{{.UserID}}{{.Type}}{{.Value}}{{.Primary}}{{.Verified}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(authentifiedUser{}, "authentified_user", false, "id"))
	gorpmapping.Register(gorpmapping.New(userContact{}, "user_contact", true, "id"))
}
