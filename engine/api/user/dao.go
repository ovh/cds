package user

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.AuthentifiedUser, error) {
	us := []authentifiedUser{}

	if err := gorpmapping.GetAll(ctx, db, q, &us); err != nil {
		return nil, sdk.WrapError(err, "cannot get authentified users")
	}

	// Check signature of data, if invalid do not return it
	verifiedUsers := make([]*sdk.AuthentifiedUser, 0, len(us))
	for i := range us {
		isValid, err := gorpmapping.CheckSignature(us[i], us[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error("user.getAll> user %s (%s) data corrupted", us[i].Username, us[i].ID)
			continue
		}
		verifiedUsers = append(verifiedUsers, &us[i].AuthentifiedUser)
	}

	if len(verifiedUsers) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, verifiedUsers...); err != nil {
				return nil, err
			}
		}
	}

	aus := make([]sdk.AuthentifiedUser, len(verifiedUsers))
	for i := range verifiedUsers {
		aus[i] = *verifiedUsers[i]
	}

	return aus, nil
}

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.AuthentifiedUser, error) {
	var u authentifiedUser

	found, err := gorpmapping.Get(ctx, db, q, &u)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get user")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrUserNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(u, u.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error("user.get> user %s (%s) data corrupted", u.Username, u.ID)
		return nil, nil
	}

	au := u.AuthentifiedUser

	for i := range opts {
		if err := opts[i](ctx, db, &au); err != nil {
			return nil, err
		}
	}

	return &au, nil
}

// LoadAll returns all users from database.
func LoadAll(ctx context.Context, db gorp.SqlExecutor, opts ...LoadOptionFunc) ([]sdk.AuthentifiedUser, error) {
	query := gorpmapping.NewQuery("SELECT * FROM authentified_user")
	return getAll(ctx, db, query, opts...)
}

// LoadAllByIDs returns users from database for given ids.
func LoadAllByIDs(ctx context.Context, db gorp.SqlExecutor, ids []string, opts ...LoadOptionFunc) ([]sdk.AuthentifiedUser, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM authentified_user
    WHERE id = ANY(string_to_array($1, ',')::text[])
  `).Args(gorpmapping.IDStringsToQueryString(ids))
	return getAll(ctx, db, query, opts...)
}

// LoadByID returns a user from database for given id.
func LoadByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadOptionFunc) (*sdk.AuthentifiedUser, error) {
	query := gorpmapping.NewQuery("SELECT * FROM authentified_user WHERE id = $1").Args(id)
	return get(ctx, db, query, opts...)
}

// LoadByUsername returns a user from database for given username.
func LoadByUsername(ctx context.Context, db gorp.SqlExecutor, username string, opts ...LoadOptionFunc) (*sdk.AuthentifiedUser, error) {
	query := gorpmapping.NewQuery("SELECT * FROM authentified_user WHERE username = $1").Args(username)
	return get(ctx, db, query, opts...)
}

// Count users in database.
func Count(db gorp.SqlExecutor) (int, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(id) FROM authentified_user").Scan(&count); err != nil {
		return 0, sdk.WithStack(err)
	}
	return count, nil
}

// Insert a user in database.
func Insert(db gorp.SqlExecutor, au *sdk.AuthentifiedUser) error {
	au.ID = sdk.UUID()
	au.DateCreation = time.Now()
	u := authentifiedUser{AuthentifiedUser: *au}
	if err := gorpmapping.InsertAndSign(db, &u); err != nil {
		return err
	}
	*au = u.AuthentifiedUser

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

// Update a user in database.
func Update(db gorp.SqlExecutor, au *sdk.AuthentifiedUser) error {
	u := authentifiedUser{AuthentifiedUser: *au}
	if err := gorpmapping.UpdatetAndSign(db, &u); err != nil {
		return sdk.WrapError(err, "unable to update authentified user with id: %s", au.ID)
	}
	*au = u.AuthentifiedUser
	return nil
}

// DeleteByID a user in database.
func DeleteByID(db gorp.SqlExecutor, id string) error {
	u, err := LoadByID(context.Background(), db, id)
	if err != nil {
		return err
	}
	if u == nil {
		return sdk.WrapError(sdk.ErrNotFound, "cannot delete not exiting authentified user with id %s", id)
	}

	// TODO: Delete user group

	_, err = db.Delete(u)
	return sdk.WithStack(err)
}

func InsertContact(db gorp.SqlExecutor, c *sdk.UserContact) error {
	dbc := userContact{UserContact: *c}
	if err := gorpmapping.InsertAndSign(db, &dbc); err != nil {
		return err
	}
	c.ID = dbc.ID
	return nil
}

func UpdateContact(db gorp.SqlExecutor, c *sdk.UserContact) error {
	dbc := userContact{UserContact: *c}
	if err := gorpmapping.UpdatetAndSign(db, &dbc); err != nil {
		return err
	}
	return nil
}

func DeleteContact(db gorp.SqlExecutor, c *sdk.UserContact) error {
	dbc := userContact{UserContact: *c}
	if err := gorpmapping.Delete(db, &dbc); err != nil {
		return err
	}
	return nil
}
