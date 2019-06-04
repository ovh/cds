package localauthentication

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func Insert(db gorp.SqlExecutor, u *sdk.UserLocalAuthentication) error {
	dbUser := userLocalAuthentication{
		UserLocalAuthentication: *u,
		EncryptedPassword:       nil,
	}
	if err := gorpmapping.Encrypt(u.ClearPassword, &dbUser.EncryptedPassword, []byte(u.UserID)); err != nil {
		return err
	}

	dbUser.ClearPassword = ""
	err := gorpmapping.InsertAndSign(db, &dbUser)
	return sdk.WithStack(err)
}

func Update(db gorp.SqlExecutor, u *sdk.UserLocalAuthentication) error {
	dbUser := userLocalAuthentication{
		UserLocalAuthentication: *u,
		EncryptedPassword:       nil,
	}
	if err := gorpmapping.Encrypt(u.ClearPassword, &dbUser.EncryptedPassword, []byte(u.UserID)); err != nil {
		return err
	}
	err := gorpmapping.UpdatetAndSign(db, &dbUser)
	return sdk.WithStack(err)
}

func Authentify(ctx context.Context, db gorp.SqlExecutor, username, password string) (bool, error) {
	u, err := user.LoadUserByUsername(ctx, db, username)
	if err != nil {
		return false, err
	}

	dbLocalAuth, err := unsafeLoadByID(db, u.ID)
	if err != nil {
		return false, err
	}

	isValid, err := gorpmapping.CheckSignature(db, &dbLocalAuth)
	if err != nil {
		return false, err
	}
	if !isValid {
		return false, sdk.WithStack(sdk.ErrCorruptedData)
	}

	if err := gorpmapping.Decrypt(dbLocalAuth.EncryptedPassword, &dbLocalAuth.ClearPassword, []byte(u.ID)); err != nil {
		return false, err
	}

	var authentified = dbLocalAuth.ClearPassword == password

	return authentified, nil
}

func LoadByID(db gorp.SqlExecutor, id string) (*sdk.UserLocalAuthentication, error) {
	u, err := unsafeLoadByID(db, id)
	if err != nil {
		return nil, err
	}
	isValid, err := gorpmapping.CheckSignature(db, &u)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, sdk.WithStack(sdk.ErrCorruptedData)
	}

	var au = u.UserLocalAuthentication
	return &au, nil
}

func unsafeLoadByID(db gorp.SqlExecutor, id string) (userLocalAuthentication, error) {
	var u userLocalAuthentication
	query := "select * from user_local_authentication where user_id = $1"
	if err := db.SelectOne(&u, query, id); err != nil {
		return u, sdk.WithStack(err)
	}
	return u, nil
}

func Delete(db gorp.SqlExecutor, id string) error {
	dbUser, err := unsafeLoadByID(db, id)
	if err != nil {
		return sdk.WithStack(err)
	}
	_, err = db.Delete(&dbUser)
	return sdk.WithStack(err)
}
