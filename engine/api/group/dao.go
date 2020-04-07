package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.Group, error) {
	pgs := []*group{}

	if err := gorpmapping.GetAll(ctx, db, q, &pgs); err != nil {
		return nil, sdk.WrapError(err, "cannot get groups")
	}

	var gs []*sdk.Group
	for i := range pgs {
		isValid, err := gorpmapping.CheckSignature(pgs[i], pgs[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "group.get> group %d data corrupted", pgs[i].ID)
			continue
		}

		gs = append(gs, &pgs[i].Group)
	}

	if len(pgs) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, gs...); err != nil {
				return nil, err
			}
		}
	}

	var result = make([]sdk.Group, len(gs))
	for i := range gs {
		result[i] = *gs[i]
	}

	return result, nil
}

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.Group, error) {
	var groupDB = group{}
	found, err := gorpmapping.Get(ctx, db, q, &groupDB)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get group")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(groupDB, groupDB.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "group.get> group %d data corrupted", groupDB.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	g := groupDB.Group

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

// LoadAllByUserID returns all groups from database for given user id.
func LoadAllByUserID(ctx context.Context, db gorp.SqlExecutor, userID string, opts ...LoadOptionFunc) (sdk.Groups, error) {
	query := gorpmapping.NewQuery(`
    SELECT "group".*
    FROM "group"
	JOIN "group_authentified_user" ON "group".id = "group_authentified_user".group_id
    WHERE "group_authentified_user".authentified_user_id = $1
    ORDER BY "group".name
  `).Args(userID)
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
func Insert(ctx context.Context, db gorp.SqlExecutor, g *sdk.Group) error {
	grp := *g
	var groupDB = group{
		Group: grp,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &groupDB); err != nil {
		return err
	}
	g.ID = groupDB.ID
	return nil
}

// Update given group into database.
func Update(ctx context.Context, db gorp.SqlExecutor, g *sdk.Group) error {
	grp := *g
	var groupDB = group{
		Group: grp,
	}
	return sdk.WrapError(gorpmapping.UpdateAndSign(ctx, db, &groupDB), "unable to update group %s", g.Name)
}

// delete given group from database.
func deleteDB(db gorp.SqlExecutor, g *sdk.Group) error {
	grp := *g
	var groupDB = group{
		Group: grp,
	}
	return sdk.WrapError(gorpmapping.Delete(db, &groupDB), "unable to delete group %s", g.Name)
}
