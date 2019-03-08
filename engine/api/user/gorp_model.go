package user

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

type authentifiedUser sdk.AuthentifiedUser
type persistentSessionToken sdk.UserToken

func init() {
	gorpmapping.Register(gorpmapping.New(persistentSessionToken{}, "user_persistent_session", false, "token"))
	gorpmapping.Register(gorpmapping.New(authentifiedUser{}, "authentified_user", false, "id"))
}

func (a *authentifiedUser) PostGet(db gorp.SqlExecutor) error {
	isValid, err := gorpmapping.CheckSignature(db, a)
	if err != nil {
		return err
	}
	if !isValid {
		return sdk.WithStack(sdk.ErrInvalidUser)
	}
	return nil
}
