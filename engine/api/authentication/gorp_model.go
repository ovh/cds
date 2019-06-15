package authentication

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type authConsumer struct {
	sdk.AuthConsumer
	gorpmapping.SignedEntity
}

type authSession struct {
	sdk.AuthSession
	gorpmapping.SignedEntity
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(authConsumer{}, "auth_consumer", false, "id"),
		gorpmapping.New(authSession{}, "auth_session", false, "id"),
	)
}
