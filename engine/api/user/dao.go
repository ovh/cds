package user

import (
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

// GetDeprecatedUser temporary code
func GetDeprecatedUser(db gorp.SqlExecutor, u *sdk.AuthentifiedUser) (*sdk.User, error) {
	oldUserID, err := db.SelectInt("select user_id from authentified_user_migration where authentified_user_id = $1", u.ID)
	if err != nil {
		return nil, err
	}
	oldUser, err := loadUserWithoutAuthByID(db, oldUserID)
	if err != nil {
		return nil, err
	}
	return oldUser, nil
}

func LoadByOldUserID(db gorp.SqlExecutor, id int64) (*sdk.AuthentifiedUser, error) {
	oldUserID, err := db.SelectStr("select user_id from authentified_user_migration where authentified_user_id = $1", id)
	if err != nil {
		return nil, err
	}
	return LoadUserByID(db, oldUserID)
}

func LoadUserByID(db gorp.SqlExecutor, id string) (*sdk.AuthentifiedUser, error) {
	u, err := unsafeLoadUserByID(db, id)
	if err != nil {
		return nil, err
	}
	isValid, err := gorpmapping.CheckSignature(db, &u)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, fmt.Errorf("corrupted data")
	}

	var au = sdk.AuthentifiedUser(u)
	return &au, nil
}

func unsafeLoadUserByID(db gorp.SqlExecutor, id string) (authentifiedUser, error) {
	var u authentifiedUser
	query := "select * from authentified_user where id = $1"
	if err := db.SelectOne(&u, query, id); err != nil {
		return u, sdk.WithStack(err)
	}
	return u, nil
}

func LoadUserByEmail(db gorp.SqlExecutor, email string) (*sdk.AuthentifiedUser, error) {
	u, err := unsafeLoadUserByEmail(db, email)
	if err != nil {
		return nil, err
	}
	isValid, err := gorpmapping.CheckSignature(db, &u)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, fmt.Errorf("corrupted data")
	}

	var au = sdk.AuthentifiedUser(u)
	return &au, nil
}

func unsafeLoadUserByEmail(db gorp.SqlExecutor, email string) (authentifiedUser, error) {
	var u authentifiedUser
	query := "select * from authentified_user where email = $1"
	if err := db.SelectOne(&u, query, email); err != nil {
		return u, sdk.WithStack(err)
	}
	return u, nil
}

func LoadUserByUsername(db gorp.SqlExecutor, username string) (*sdk.AuthentifiedUser, error) {
	u, err := unsafeLoadUserByUsername(db, username)
	if err != nil {
		return nil, err
	}
	isValid, err := gorpmapping.CheckSignature(db, &u)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, fmt.Errorf("corrupted data")
	}

	var au = sdk.AuthentifiedUser(u)
	return &au, nil
}

func unsafeLoadUserByUsername(db gorp.SqlExecutor, username string) (authentifiedUser, error) {
	var u authentifiedUser
	query := "select * from authentified_user where username = $1"
	if err := db.SelectOne(&u, query, username); err != nil {
		return u, sdk.WithStack(err)
	}
	return u, nil
}

func Insert(db gorp.SqlExecutor, u *sdk.AuthentifiedUser) error {
	if u.ID == "" {
		u.ID = sdk.UUID()
	}
	var dbUser = authentifiedUser(*u)
	if err := gorpmapping.InsertAndSign(db, &dbUser); err != nil {
		return err
	}
	return nil
}

func Update(db gorp.SqlExecutor, u *sdk.AuthentifiedUser) error {
	var dbUser = authentifiedUser(*u)
	if err := gorpmapping.UpdatetAndSign(db, &dbUser); err != nil {
		return err
	}
	return nil
}

func Delete(db gorp.SqlExecutor, id string) error {
	dbUser, err := unsafeLoadUserByID(db, id)
	if err != nil {
		return sdk.WithStack(err)
	}
	_, err = db.Delete(&dbUser)
	return sdk.WithStack(err)
}

func InsertContact(db gorp.SqlExecutor, c *sdk.UserContact) error {
	dbc := userContact(*c)
	if err := gorpmapping.InsertAndSign(db, dbc); err != nil {
		return err
	}
	c.ID = dbc.ID
	return nil
}

func UpdateContact(db gorp.SqlExecutor, c *sdk.UserContact) error {
	dbc := userContact(*c)
	if err := gorpmapping.UpdatetAndSign(db, dbc); err != nil {
		return err
	}
	return nil
}

func DeleteContact(db gorp.SqlExecutor, c *sdk.UserContact) error {
	dbc := userContact(*c)
	if err := gorpmapping.Delete(db, dbc); err != nil {
		return err
	}
	return nil
}

func LoadContacts(db gorp.SqlExecutor, u *sdk.AuthentifiedUser) ([]sdk.UserContact, error) {
	query := "select * from user_contact where user_id = $1 order by id asc"
	var contacts []sdk.UserContact
	var dbContacts []userContact
	if _, err := db.Select(&dbContacts, query, u.ID); err != nil {
		return nil, err
	}
	for i := range dbContacts {
		// TODO do not return if any error
		ok, err := gorpmapping.CheckSignature(db, dbContacts[i])
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("corrupted data")
		}
		contacts = append(contacts, sdk.UserContact(dbContacts[i]))
	}
	return contacts, nil
}
