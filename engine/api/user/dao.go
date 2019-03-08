package user

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

func LoadUserByID(db gorp.SqlExecutor, id string) (*sdk.AuthentifiedUser, error) {
	var u sdk.AuthentifiedUser
	query := "select * from authentified_user where id = $1"
	if err := db.SelectOne(&u, query, id); err != nil {
		return nil, err
	}
	return nil, nil
}

func Insert(db gorp.SqlExecutor, u *sdk.AuthentifiedUser) error {
	if u.ID == "" {
		u.ID = sdk.UUID()
	}

	if err := gorpmapping.InsertAndSign(db, u); err != nil {
		return err
	}
	return nil
}

func Update(db gorp.SqlExecutor, u *sdk.AuthentifiedUser) error {
	if err := gorpmapping.UpdatetAndSign(db, u); err != nil {
		return err
	}
	return nil
}
