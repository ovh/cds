package accesstoken

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getSessions(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadSessionOptionFunc) ([]sdk.AuthSession, error) {
	pSessions := []*sdk.AuthSession{}

	if err := gorpmapping.GetAll(ctx, db, q, &pSessions); err != nil {
		return nil, sdk.WrapError(err, "cannot get auth sessions")
	}
	if len(pSessions) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, pSessions...); err != nil {
				return nil, err
			}
		}
	}

	sessions := make([]sdk.AuthSession, len(pSessions))
	for i := range pSessions {
		sessions[i] = *pSessions[i]
	}

	return sessions, nil
}

func getSession(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadSessionOptionFunc) (*sdk.AuthSession, error) {
	var session sdk.AuthSession

	found, err := gorpmapping.Get(ctx, db, q, &session)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get auth session")
	}
	if !found {
		return nil, nil
	}

	for i := range opts {
		if err := opts[i](ctx, db, &session); err != nil {
			return nil, err
		}
	}

	return &session, nil
}

// LoadSessionByID returns an auth session from database.
func LoadSessionByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...LoadSessionOptionFunc) (*sdk.AuthSession, error) {
	query := gorpmapping.NewQuery("SELECT * FROM auth_session WHERE id = $1").Args(id)
	return getSession(ctx, db, query, opts...)
}

// LoadSessionsByUserID returns all auth sessions created by a user.
/*func LoadSessionsByUserID(ctx context.Context, db gorp.SqlExecutor, userID string, opts ...LoadOptionFunc) ([]sdk.AccessToken, error) {
	query := gorpmapping.NewQuery("SELECT * FROM access_token WHERE user_id = $1 ORDER BY created ASC").Args(userID)
	return getAll(ctx, db, query, opts...)
}*/

// LoadAllByGroupID returns all tokens associated to a group.
/*func LoadAllByGroupID(ctx context.Context, db gorp.SqlExecutor, groupID int64, opts ...LoadOptionFunc) ([]sdk.AccessToken, error) {
	query := gorpmapping.NewQuery(`
    SELECT access_token.*
    FROM access_token
    JOIN access_token_group ON access_token.id = access_token_group.access_token_id
    WHERE access_token_group.group_id = $1
    ORDER BY created asc
  `).Args(groupID)
	return getAll(ctx, db, query, opts...)
}*/

// InsertSession in database.
func InsertSession(db gorp.SqlExecutor, s *sdk.AuthSession) error {
	if err := gorpmapping.Insert(db, s); err != nil {
		return sdk.WrapError(err, "unable to insert auth session")
	}
	return nil
}

// UpdateSession in database.
func UpdateSession(db gorp.SqlExecutor, s *sdk.AuthSession) error {
	if err := gorpmapping.Update(db, s); err != nil {
		return sdk.WrapError(err, "unable to update auth session with id: %s", s.ID)
	}
	return nil
}

// DeleteSessionByID removes a auth session in database for given id.
func DeleteSessionByID(db gorp.SqlExecutor, id string) error {
	_, err := db.Exec("DELETE FROM auth_session WHERE id = $1", id)
	return sdk.WrapError(err, "unable to delete auth session with id %s", id)
}
