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
	_ = []interface{}{u.ID, u.Username, u.Fullname, u.Ring, u.Created}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Username}}{{.Fullname}}{{.Ring}}{{printDate .Created}}",
	}
}

type userContact struct {
	sdk.UserContact
	gorpmapper.SignedEntity
}

func (c userContact) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{c.ID, c.UserID, c.Type, c.Value, c.Primary, c.Verified}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.UserID}}{{.Type}}{{.Value}}{{.Primary}}{{.Verified}}",
	}
}

type Organization struct {
	ID                 int64  `db:"id"`
	AuthentifiedUserID string `db:"authentified_user_id"`
	Organization       string `db:"organization"`
	gorpmapper.SignedEntity
}

func (o Organization) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{o.ID, o.AuthentifiedUserID, o.Organization} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{print .ID}}{{.AuthentifiedUserID}}{{.Organization}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(authentifiedUser{}, "authentified_user", false, "id"))
	gorpmapping.Register(gorpmapping.New(userContact{}, "user_contact", true, "id"))
	gorpmapping.Register(gorpmapping.New(Organization{}, "authentified_user_organization", true, "id"))
}
