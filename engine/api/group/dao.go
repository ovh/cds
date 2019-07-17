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
			if err := opts[i](ctx, db, pgs...); err != nil {
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

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.Group, error) {
	var g sdk.Group

	found, err := gorpmapping.Get(ctx, db, q, &g)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get group")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	for i := range opts {
		if err := opts[i](ctx, db, &g); err != nil {
			return nil, err
		}
	}

	return &g, nil
}

// LoadAll returns all groups from database.
func LoadAll(ctx context.Context, db gorp.SqlExecutor, opts ...LoadOptionFunc) (sdk.Groups, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM "group"
    ORDER BY "group".name
  `)
	return getAll(ctx, db, query, opts...)
}

// LoadAllByIDs returns all groups from database for given ids.
func LoadAllByIDs(ctx context.Context, db gorp.SqlExecutor, ids []int64, opts ...LoadOptionFunc) (sdk.Groups, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM "group"
    WHERE id = ANY(string_to_array($1, ',')::int[])
  `).Args(gorpmapping.IDsToQueryString(ids))
	return getAll(ctx, db, query, opts...)
}

// LoadAllByDeprecatedUserID returns all groups from database for given user id.
func LoadAllByDeprecatedUserID(ctx context.Context, db gorp.SqlExecutor, deprecatedUserID int64, opts ...LoadOptionFunc) ([]sdk.Group, error) {
	query := gorpmapping.NewQuery(`
    SELECT "group".*
    FROM "group"
		JOIN "group_user" ON "group".id = "group_user".group_id
    WHERE "group_user".user_id = $1
    ORDER BY "group".name
  `).Args(deprecatedUserID)
	return getAll(ctx, db, query, opts...)
}

// LoadByName retrieves a group by name from database.
func LoadByName(ctx context.Context, db gorp.SqlExecutor, name string, opts ...LoadOptionFunc) (*sdk.Group, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM "group"
    WHERE "group".name = $1
  `).Args(name)
	return get(ctx, db, query, opts...)
}

// LoadByID retrieves group from database by id.
func LoadByID(ctx context.Context, db gorp.SqlExecutor, id int64, opts ...LoadOptionFunc) (*sdk.Group, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM "group"
    WHERE "group".id = $1
  `).Args(id)
	return get(ctx, db, query, opts...)
}

// Insert given group into database.
func Insert(db gorp.SqlExecutor, g *sdk.Group) error {
	return sdk.WrapError(gorpmapping.Insert(db, g), "unable to insert group %s", g.Name)
}

// Update given group into database.
func Update(db gorp.SqlExecutor, g *sdk.Group) error {
	return sdk.WrapError(gorpmapping.Update(db, g), "unable to update group %s", g.Name)
}

// delete given group from database.
func deleteDB(db gorp.SqlExecutor, g *sdk.Group) error {
	return sdk.WrapError(gorpmapping.Delete(db, g), "unable to delete group %s", g.Name)
}
