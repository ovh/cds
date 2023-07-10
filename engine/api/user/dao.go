package user

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
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
			return nil, sdk.WrapError(err, "error when checking signature for user %s", us[i].ID)
		}
		if !isValid {
			log.Error(ctx, "user.getAll> user %s (%s) data corrupted", us[i].Username, us[i].ID)
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

func Get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.AuthentifiedUser, error) {
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
		log.Error(ctx, "user.get> user %s (%s) data corrupted", u.Username, u.ID)
		return nil, sdk.WithStack(sdk.ErrUserNotFound)
	}

	au := u.AuthentifiedUser

	for i := range opts {
		if err := opts[i](ctx, db, &au); err != nil {
			return nil, err
		}
	}

	return &au, nil
}

func LoadUsersWithoutOrganization(ctx context.Context, db gorp.SqlExecutor) ([]sdk.AuthentifiedUser, error) {
	query := gorpmapping.NewQuery(`
		SELECT au.* FROM authentified_user au
		LEFT JOIN authentified_user_organization auo ON auo.authentified_user_id = au.id
		WHERE auo.authentified_user_id is null`)
	return getAll(ctx, db, query)
}

// LoadAll returns all users from database.
func LoadAll(ctx context.Context, db gorp.SqlExecutor, opts ...LoadOptionFunc) (sdk.AuthentifiedUsers, error) {
	query := gorpmapping.NewQuery("SELECT * FROM authentified_user")
	return getAll(ctx, db, query, opts...)
}

// LoadAllByRing returns users from database for given ids.
func LoadAllByRing(ctx context.Context, db gorp.SqlExecutor, ring string, opts ...LoadOptionFunc) (sdk.AuthentifiedUsers, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM authentified_user
    WHERE ring = $1
  `).Args(ring)
	return getAll(ctx, db, query, opts...)
}

// LoadAllByIDs returns users from database for given ids.
func LoadAllByIDs(ctx context.Context, db gorp.SqlExecutor, ids []string, opts ...LoadOptionFunc) (sdk.AuthentifiedUsers, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM authentified_user
    WHERE id = ANY(string_to_array($1, ',')::text[])
  `).Args(gorpmapping.IDStringsToQueryString(ids))
	return getAll(ctx, db, query, opts...)
}

// LoadByID returns a user from database for given id.
func LoadByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadOptionFunc) (*sdk.AuthentifiedUser, error) {
	ctx, next := telemetry.Span(ctx, "user.LoadByID")
	defer next()
	query := gorpmapping.NewQuery("SELECT * FROM authentified_user WHERE id = $1").Args(id)
	return Get(ctx, db, query, opts...)
}

// LoadByUsername returns a user from database for given username.
func LoadByUsername(ctx context.Context, db gorp.SqlExecutor, username string, opts ...LoadOptionFunc) (*sdk.AuthentifiedUser, error) {
	query := gorpmapping.NewQuery("SELECT * FROM authentified_user WHERE username = $1").Args(username)
	return Get(ctx, db, query, opts...)
}

// CountAdmin admin users in database.
func CountAdmin(db gorp.SqlExecutor) (int64, error) {
	count, err := db.SelectInt("SELECT COUNT(id) FROM authentified_user WHERE ring = 'ADMIN'")
	if err != nil {
		return 0, sdk.WithStack(err)
	}
	return count, nil
}

// Insert a user in database.
func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, au *sdk.AuthentifiedUser) error {
	if err := sdk.IsValidUsername(au.Username); err != nil {
		return err
	}

	au.ID = sdk.UUID()
	au.Created = time.Now()
	u := authentifiedUser{AuthentifiedUser: *au}
	if err := gorpmapping.InsertAndSign(ctx, db, &u); err != nil {
		return err
	}
	*au = u.AuthentifiedUser

	return nil
}

// Update a user in database.
func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, au *sdk.AuthentifiedUser) error {
	if err := sdk.IsValidUsername(au.Username); err != nil {
		return err
	}

	u := authentifiedUser{AuthentifiedUser: *au}
	if err := gorpmapping.UpdateAndSign(ctx, db, &u); err != nil {
		return sdk.WrapError(err, "unable to update authentified user with id: %s", au.ID)
	}
	*au = u.AuthentifiedUser
	return nil
}

// DeleteByID a user in database.
func DeleteByID(db gorp.SqlExecutor, id string) error {
	_, err := db.Exec("DELETE FROM authentified_user WHERE id = $1", id)
	return sdk.WithStack(err)
}
