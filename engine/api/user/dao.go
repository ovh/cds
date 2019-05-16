package user

import (
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk/log"

	"github.com/ovh/cds/sdk"
)

// GetDeprecatedUser temporary code
func GetDeprecatedUser(db gorp.SqlExecutor, u *sdk.AuthentifiedUser) (*sdk.User, error) {
	oldUserID, err := db.SelectInt("select user_id from authentified_user_migration where authentified_user_id = $1", u.ID)
	if err != nil {
		return nil, sdk.WrapError(sdk.ErrInvalidUser, "unable to load user_id from authentified_user_migration where authentified_user_id = %s: %v", u.ID, err)
	}
	oldUser, err := deprecatedLoadUserWithoutAuthByID(db, oldUserID)
	if err != nil {
		return nil, err
	}
	return oldUser, nil
}

func LoadByOldUserID(db gorp.SqlExecutor, id int64, opts ...LoadOptionFunc) (*sdk.AuthentifiedUser, error) {
	oldUserID, err := db.SelectStr("select authentified_user_id from authentified_user_migration where user_id = $1", id)
	if err != nil {
		return nil, sdk.WrapError(sdk.ErrInvalidUser, "unable to load authentified_user_id from authentified_user_migration where user_id = %d: %v", id, err)
	}
	return LoadUserByID(db, oldUserID, opts...)
}

func LoadUserByID(db gorp.SqlExecutor, id string, opts ...LoadOptionFunc) (*sdk.AuthentifiedUser, error) {
	u, err := unsafeLoadUserByID(db, id)
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

	var au = sdk.AuthentifiedUser(u)
	if len(opts) == 0 {
		opts = []LoadOptionFunc{
			LoadOptions.WithOldUserStruct, LoadOptions.WithContacts,
		}
	}
	for i := range opts {
		if err := opts[i](db, &au); err != nil {
			return nil, err
		}
	}

	return &au, nil
}

func unsafeLoadUserByID(db gorp.SqlExecutor, id string) (authentifiedUser, error) {
	var u authentifiedUser
	query := gorpmapping.NewQuery("select * from authentified_user where id = $1").Args(id)
	if has, err := gorpmapping.Get(db, query, &u); err != nil {
		return u, sdk.WrapError(err, "unable to load user by id %s", id)
	} else if !has {
		return u, sdk.WrapError(sdk.ErrInvalidUser, "unable to load user by id %s", id)
	}
	return u, nil
}

func LoadUserByEmail(db gorp.SqlExecutor, email string, opts ...LoadOptionFunc) (*sdk.AuthentifiedUser, error) {
	u, err := unsafeLoadUserByEmail(db, email)
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

	var au = sdk.AuthentifiedUser(u)

	for i := range opts {
		if err := opts[i](db, &au); err != nil {
			return nil, err
		}
	}

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

func LoadUserByUsername(db gorp.SqlExecutor, username string, opts ...LoadOptionFunc) (*sdk.AuthentifiedUser, error) {
	u, err := unsafeLoadUserByUsername(db, username)
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

	var au = sdk.AuthentifiedUser(u)

	for i := range opts {
		if err := opts[i](db, &au); err != nil {
			return nil, err
		}
	}

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

type LoadOptionFunc func(gorp.SqlExecutor, ...*sdk.AuthentifiedUser) error

var LoadOptions = struct {
	WithContacts      LoadOptionFunc
	WithOldUserStruct LoadOptionFunc
}{
	WithContacts:      LoadContacts,
	WithOldUserStruct: LoadDeprecatedUser, // TODO: will be removed
}

func LoadAll(db gorp.SqlExecutor, opts ...LoadOptionFunc) ([]sdk.AuthentifiedUser, error) {
	var dbUsers []authentifiedUser
	query := gorpmapping.NewQuery("select * from authentified_user")
	if err := gorpmapping.GetAll(db, query, &dbUsers); err != nil {
		return nil, err
	}

	// TODO: options
	var users []sdk.AuthentifiedUser
	for i := range dbUsers {
		u := &dbUsers[i]
		isValid, err := gorpmapping.CheckSignature(db, u)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error("user.LoadAll> user %s (%s) data corrupted", u.Username, u.ID)
			continue
		}

		au := sdk.AuthentifiedUser(*u)
		for i := range opts {
			if err := opts[i](db, &au); err != nil {
				return nil, err
			}
		}
		users = append(users, au)
	}

	return users, nil
}

func Insert(db gorp.SqlExecutor, u *sdk.AuthentifiedUser) error {
	if u.ID == "" {
		u.ID = sdk.UUID()
	}
	var dbUser = authentifiedUser(*u)
	if err := gorpmapping.InsertAndSign(db, &dbUser); err != nil {
		return err
	}

	// TODO: refactor this when authenticatedUser.group will replace user.group
	oldUser := &sdk.User{
		Admin:    u.Ring == sdk.UserRingAdmin,
		Email:    "no-reply-" + u.Username + "@corp.ovh.com",
		Username: u.Username,
		Origin:   "local",
		Fullname: u.Fullname,
		Auth: sdk.Auth{
			EmailVerified:  true,
			HashedPassword: sdk.RandomString(12),
		},
	}
	if err := insertUser(db, oldUser, &oldUser.Auth); err != nil {
		return sdk.WrapError(err, "unable to insert old user for authenticated_user.id=%s", u.ID)
	}
	query := "INSERT INTO authentified_user_migration(authentified_user_id, user_id) VALUES($1, $2)"
	if _, err := db.Exec(query, u.ID, oldUser.ID); err != nil {
		return sdk.WrapError(err, "unable to insert into table authentified_user_migration")
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

	// TODO: Delete user group

	_, err = db.Delete(&dbUser)
	return sdk.WithStack(err)
}

func InsertContact(db gorp.SqlExecutor, c *sdk.UserContact) error {
	dbc := userContact(*c)
	if err := gorpmapping.InsertAndSign(db, &dbc); err != nil {
		return err
	}
	c.ID = dbc.ID
	return nil
}

func UpdateContact(db gorp.SqlExecutor, c *sdk.UserContact) error {
	dbc := userContact(*c)
	if err := gorpmapping.UpdatetAndSign(db, &dbc); err != nil {
		return err
	}
	return nil
}

func DeleteContact(db gorp.SqlExecutor, c *sdk.UserContact) error {
	dbc := userContact(*c)
	if err := gorpmapping.Delete(db, &dbc); err != nil {
		return err
	}
	return nil
}

func LoadContacts(db gorp.SqlExecutor, u ...*sdk.AuthentifiedUser) error {
	usersID := sdk.AuthentifiedUsersToIDs(u)
	query := gorpmapping.NewQuery(
		"select * from user_contact where user_id = ANY(string_to_array($1, ',')::text[]) order by id asc",
	).Args(strings.Join(usersID, ","))

	var dbContacts []userContact
	if err := gorpmapping.GetAll(db, query, &dbContacts); err != nil {
		return err
	}

	mapUsers := make(map[string][]sdk.UserContact, len(dbContacts))
	for i := range dbContacts {
		if _, ok := mapUsers[dbContacts[i].UserID]; !ok {
			mapUsers[dbContacts[i].UserID] = make([]sdk.UserContact, 0, len(dbContacts))
		}

		// TODO do not return if any error
		ok, err := gorpmapping.CheckSignature(db, dbContacts[i])
		if err != nil {
			return err
		}
		if !ok {
			return sdk.WithStack(sdk.ErrCorruptedData)
		}

		mapUsers[dbContacts[i].UserID] = append(mapUsers[dbContacts[i].UserID], sdk.UserContact(dbContacts[i]))
	}

	for i := range u {
		u[i].Contacts = mapUsers[u[i].ID]
	}

	return nil
}

func LoadDeprecatedUser(db gorp.SqlExecutor, u ...*sdk.AuthentifiedUser) error {

	for _, u := range u {
		oldUserID, err := db.SelectInt("select user_id from authentified_user_migration where authentified_user_id = $1", u.ID)
		if err != nil {
			return err
		}
		oldUser, err := deprecatedLoadUserWithoutAuthByID(db, oldUserID)
		if err != nil {
			return err
		}
		u.OldUserStruct = oldUser
	}

	return nil
}
