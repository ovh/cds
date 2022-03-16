package authentication

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type authConsumer struct {
	sdk.AuthConsumer
	gorpmapper.SignedEntity
}

func (c authConsumer) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{c.ID, c.AuthentifiedUserID, c.Type, c.Data, c.Created, c.GroupIDs, c.ScopeDetails, c.Disabled, c.ServiceName, c.ServiceType, c.ServiceRegion} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.AuthentifiedUserID}}{{print .Type}}{{print .Data}}{{printDate .Created}}{{print .GroupIDs}}{{print .ScopeDetails}}{{print .Disabled}}{{.ServiceName}}{{.ServiceType}}{{.ServiceRegion}}",
		"{{.ID}}{{.AuthentifiedUserID}}{{print .Type}}{{print .Data}}{{printDate .Created}}{{print .GroupIDs}}{{print .ScopeDetails}}{{print .Disabled}}",
	}
}

type authSession struct {
	sdk.AuthSession
	gorpmapper.SignedEntity
}

func (s authSession) Canonical() gorpmapper.CanonicalForms {
	_ = []interface{}{s.ID, s.ConsumerID, s.ExpireAt, s.Created} // Checks that fields exists at compilation
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.ConsumerID}}{{printDate .ExpireAt}}{{printDate .Created}}",
	}
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(authConsumer{}, "auth_consumer", false, "id"),
		gorpmapping.New(authSession{}, "auth_session", false, "id"),
	)
}
