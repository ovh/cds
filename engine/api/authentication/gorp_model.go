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
	_ = []interface{}{c.ID, c.AuthentifiedUserID, c.Type, c.Data, c.Created, c.GroupIDs, c.ScopeDetails, c.Disabled} // Checks that fields exists at compilation
	return []gorpmapping.CanonicalForm{
		"{{.ID}}{{.AuthentifiedUserID}}{{print .Type}}{{print .Data}}{{printDate .Created}}{{print .GroupIDs}}{{print .ScopeDetails}}{{print .Disabled}}",
	}
}

type authSession struct {
	sdk.AuthSession
	gorpmapping.SignedEntity
}

func (s authSession) Canonical() gorpmapping.CanonicalForms {
	_ = []interface{}{s.ID, s.ConsumerID, s.ExpireAt, s.Created} // Checks that fields exists at compilation
	return []gorpmapping.CanonicalForm{
		"{{.ID}}{{.ConsumerID}}{{printDate .ExpireAt}}{{printDate .Created}}",
	}
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(authConsumer{}, "auth_consumer", false, "id"),
		gorpmapping.New(authSession{}, "auth_session", false, "id"),
	)
}
