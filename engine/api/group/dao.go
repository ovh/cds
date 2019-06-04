package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.Group, error) {
	pgs := []*sdk.Group{}

	if err := gorpmapping.GetAll(ctx, db, q, &pgs); err != nil {
		return nil, sdk.WrapError(err, "cannot get groups")
	}
	if len(pgs) > 0 {
		for i := range opts {
			if err := opts[i](db, pgs...); err != nil {
				return nil, err
			}
		}
	}

	gs := make([]sdk.Group, len(pgs))
	for i := range gs {
		gs[i] = *pgs[i]
	}

	return gs, nil
}

// LoadAllByIDs returns all groups from database for given ids.
func LoadAllByIDs(ctx context.Context, db gorp.SqlExecutor, ids []int64, opts ...LoadOptionFunc) ([]sdk.Group, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM "group"
    WHERE id = ANY(string_to_array($1, ',')::int[])
  `).Args(gorpmapping.IDsToQueryString(ids))
	return getAll(ctx, db, query, opts...)
}

// GetLinksGroupUserForUserIDs returns data from group_user table for given user ids.
func GetLinksGroupUserForUserIDs(ctx context.Context, db gorp.SqlExecutor, userIDs []int64) ([]LinkGroupUser, error) {
	ls := []LinkGroupUser{}

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM group_user
    WHERE user_id = ANY(string_to_array($1, ',')::int[])
  `).Args(gorpmapping.IDsToQueryString(userIDs))

	if err := gorpmapping.GetAll(ctx, db, query, &ls); err != nil {
		return nil, sdk.WrapError(err, "cannot get links between group and user")
	}

	return ls, nil
}
