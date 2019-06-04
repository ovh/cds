package accesstoken

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.AccessToken, error) {
	pats := []*sdk.AccessToken{}

	if err := gorpmapping.GetAll(ctx, db, q, &pats); err != nil {
		return nil, sdk.WrapError(err, "cannot get access tokens")
	}
	if len(pats) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, pats...); err != nil {
				return nil, err
			}
		}
	}

	ats := make([]sdk.AccessToken, len(pats))
	for i := range pats {
		ats[i] = *pats[i]
	}

	return ats, nil
}

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.AccessToken, error) {
	var at sdk.AccessToken

	found, err := gorpmapping.Get(ctx, db, q, &at)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get access token")
	}
	if !found {
		return nil, nil
	}

	for i := range opts {
		if err := opts[i](ctx, db, &at); err != nil {
			return nil, err
		}
	}

	return &at, nil
}

// LoadByID returns an access token from database.
func LoadByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadOptionFunc) (*sdk.AccessToken, error) {
	query := gorpmapping.NewQuery("SELECT * FROM access_token WHERE id = $1").Args(id)
	return get(ctx, db, query, opts...)
}

// LoadAllByUserID returns all tokens created by a user.
func LoadAllByUserID(ctx context.Context, db gorp.SqlExecutor, userID string, opts ...LoadOptionFunc) ([]sdk.AccessToken, error) {
	query := gorpmapping.NewQuery("SELECT * FROM access_token WHERE user_id = $1 ORDER BY created ASC").Args(userID)
	return getAll(ctx, db, query, opts...)
}

// LoadAllByGroupID returns all tokens associated to a group.
func LoadAllByGroupID(ctx context.Context, db gorp.SqlExecutor, groupID int64, opts ...LoadOptionFunc) ([]sdk.AccessToken, error) {
	query := gorpmapping.NewQuery(`
    SELECT access_token.*
    FROM access_token
    JOIN access_token_group ON access_token.id = access_token_group.access_token_id
    WHERE access_token_group.group_id = $1
    ORDER BY created asc
  `).Args(groupID)
	return getAll(ctx, db, query, opts...)
}

// Insert a new token in database
func Insert(db gorp.SqlExecutor, at *sdk.AccessToken) error {
	t := accessToken(*at)
	if err := gorpmapping.Insert(db, &t); err != nil {
		return sdk.WrapError(err, "unable to insert access token")
	}
	*at = sdk.AccessToken(t)
	return nil
}

// Update a token in database
func Update(db gorp.SqlExecutor, at *sdk.AccessToken) error {
	t := accessToken(*at)
	if err := gorpmapping.Update(db, &t); err != nil {
		return sdk.WrapError(err, "unable to update access token with id: %s", at.ID)
	}
	*at = sdk.AccessToken(t)
	return nil
}

// Delete a token in database
func Delete(db gorp.SqlExecutor, id string) error {
	at, err := LoadByID(context.Background(), db, id)
	if err != nil {
		return err
	}
	if at == nil {
		return sdk.WrapError(sdk.ErrNotFound, "cannot delete not exiting access token with id %s", id)
	}

	t := accessToken(*at)
	_, err = db.Delete(&t)
	return sdk.WithStack(err)
}

func getAccessTokenGroupsForAccessTokenIDs(ctx context.Context, db gorp.SqlExecutor, accessTokenIDs []string) ([]accessTokenGroup, error) {
	var accessTokenGroup []accessTokenGroup

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM access_token_group
    WHERE access_token_id = ANY(string_to_array($1, ',')::text[])
  `).Args(gorpmapping.IDStringsToQueryString(accessTokenIDs))

	if err := gorpmapping.GetAll(ctx, db, query, &accessTokenGroup); err != nil {
		return nil, sdk.WrapError(err, "unable to get access tokens groups")
	}

	return accessTokenGroup, nil
}
