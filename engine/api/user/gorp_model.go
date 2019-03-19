package user

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

type authentifiedUser sdk.AuthentifiedUser
type persistentSessionToken sdk.UserToken
type userContact sdk.UserContact

func init() {
	gorpmapping.Register(gorpmapping.New(persistentSessionToken{}, "user_persistent_session", false, "token"))
	gorpmapping.Register(gorpmapping.New(authentifiedUser{}, "authentified_user", false, "id"))
	gorpmapping.Register(gorpmapping.New(userContact{}, "user_contact", true, "id"))
}
