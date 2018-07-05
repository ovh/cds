package user

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

type persistentSessionToken sdk.UserToken

func init() {
	gorpmapping.Register(gorpmapping.New(persistentSessionToken{}, "user_persistent_session", false, "token"))
}
