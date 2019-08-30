package authentication

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type authConsumer struct {
	sdk.AuthConsumer
	gorpmapping.SignedEntity
}

func (c authConsumer) Canonical() gorpmapping.CanonicalForms {
	return []gorpmapping.CanonicalForm{
		"{{.ID}}{{.AuthentifiedUserID}}{{print .Type}}{{print .Data}}{{printDate .Created}}{{print .GroupIDs}}{{print .Scopes}}",
	}
}

type authSession struct {
	sdk.AuthSession
	gorpmapping.SignedEntity
}

func (s authSession) Canonical() gorpmapping.CanonicalForms {
	return []gorpmapping.CanonicalForm{
		"{{.ID}}{{.ConsumerID}}{{printDate .ExpireAt}}{{printDate .Created}}{{print .GroupIDs}}{{print .Scopes}}",
	}
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(authConsumer{}, "auth_consumer", false, "id"),
		gorpmapping.New(authSession{}, "auth_session", false, "id"),
	)
}
