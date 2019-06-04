package user

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getDeprecatedUsers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadDeprecatedUserOptionFunc) ([]sdk.User, error) {
	dus := []deprecatedUser{}

	if err := gorpmapping.GetAll(ctx, db, q, &dus); err != nil {
		return nil, sdk.WrapError(err, "cannot get deprecated users")
	}

	pus := make([]*sdk.User, len(dus))
	for i := range dus {
		pus[i] = &dus[i].Data
		pus[i].ID = dus[i].ID
		pus[i].Admin = dus[i].Admin
		pus[i].Origin = dus[i].Origin
	}

	if len(pus) > 0 {
		for i := range opts {
			if err := opts[i](db, pus...); err != nil {
				return nil, err
			}
		}
	}

	us := make([]sdk.User, len(pus))
	for i := range pus {
		us[i] = *pus[i]
	}

	return us, nil
}

func getDeprecatedUser(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadDeprecatedUserOptionFunc) (*sdk.User, error) {
	var du deprecatedUser

	found, err := gorpmapping.Get(ctx, db, q, &du)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get deprecated user")
	}
	if !found {
		return nil, nil
	}

	pu := &du.Data
	pu.ID = du.ID
	pu.Admin = du.Admin
	pu.Origin = du.Origin

	for i := range opts {
		if err := opts[i](db, pu); err != nil {
			return nil, err
		}
	}

	return pu, nil
}

// LoadDeprecatedUsersWithoutAuthByIDs returns deprecated users from database for given ids.
func LoadDeprecatedUsersWithoutAuthByIDs(ctx context.Context, db gorp.SqlExecutor, ids []int64, opts ...LoadDeprecatedUserOptionFunc) ([]sdk.User, error) {
	query := gorpmapping.NewQuery(`
    SELECT id, username, admin, data, origin
    FROM "user"
    WHERE id = ANY(string_to_array($1, ',')::int[])
  `).Args(gorpmapping.IDsToQueryString(ids))
	return getDeprecatedUsers(ctx, db, query, opts...)
}

// LoadDeprecatedUserWithoutAuthByID returns deprecated user from database for given id.
func LoadDeprecatedUserWithoutAuthByID(ctx context.Context, db gorp.SqlExecutor, id int64, opts ...LoadDeprecatedUserOptionFunc) (*sdk.User, error) {
	query := gorpmapping.NewQuery(`
    SELECT id, username, admin, data, origin
    FROM "user"
    WHERE id = $1
  `).Args(id)
	return getDeprecatedUser(ctx, db, query, opts...)
}
