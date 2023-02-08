package link

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type dbUserLink struct {
	sdk.UserLink
	gorpmapper.SignedEntity
}

func (ul dbUserLink) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{ul.ID, ul.AuthentifiedUserID, ul.Type, ul.ExternalID}
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.AuthentifiedUserID}}{{.Type}}{{.ExternalID}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(dbUserLink{}, "user_link", true, "id"))
}
