package user

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type authentifiedUser struct {
	sdk.AuthentifiedUser
	gorpmapping.SignedEntity
}

type persistentSessionToken sdk.UserToken

type userContact struct {
	sdk.UserContact
	gorpmapping.SignedEntity
}

type authentifiedUserMigration struct {
	AuthentifiedUserID string `db:"authentified_user_id"`
	UserID             int64  `db:"user_id"`
}

type deprecatedUser struct {
	ID       int64    `db:"id"`
	Username string   `db:"username"`
	Admin    bool     `db:"admin"`
	Origin   string   `db:"origin"`
	Data     sdk.User `db:"data"`
}

func init() {
	gorpmapping.Register(gorpmapping.New(persistentSessionToken{}, "user_persistent_session", false, "token"))
	gorpmapping.Register(gorpmapping.New(sdk.AuthentifiedUser{}, "authentified_user", false, "id"))
	gorpmapping.Register(gorpmapping.New(userContact{}, "user_contact", true, "id"))
	gorpmapping.Register(gorpmapping.New(authentifiedUserMigration{}, "authentified_user_migration", false, "authentified_user_id", "user_id"))
	gorpmapping.Register(gorpmapping.New(deprecatedUser{}, "user", true, "id"))
}
