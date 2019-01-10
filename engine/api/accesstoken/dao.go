package accesstoken

import (
	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// FindByID returns an access token from database
func FindByID(db gorp.SqlExecutor, id string) (sdk.AccessToken, error) {
	var token accessToken
	if err := db.SelectOne(&token, "select * from access_token where id = $1", id); err != nil {
		return sdk.AccessToken{}, sdk.WithStack(err)
	}

	return sdk.AccessToken(token), nil
}

// Insert a new token in database
func Insert(db gorp.SqlExecutor, token *sdk.AccessToken) error {
	dbToken := accessToken(*token)
	if err := db.Insert(&dbToken); err != nil {
		return sdk.WithStack(err)
	}
	log.Debug("access token %s inserted", token.ID)
	return nil
}

// Update a token in database
func Update(db gorp.SqlExecutor, token *sdk.AccessToken) error {
	dbToken := accessToken(*token)
	n, err := db.Update(&dbToken)
	if err != nil {
		return sdk.WithStack(err)
	}
	if n < 1 {
		return sdk.WithStack(sdk.ErrNotFound)
	}
	log.Debug("access token %s updated", token.ID)
	return nil
}

// Delete a token in database
func Delete(db gorp.SqlExecutor, token *sdk.AccessToken) error {
	dbToken := accessToken(*token)
	n, err := db.Delete(&dbToken)
	if err != nil {
		return sdk.WithStack(err)
	}
	if n < 1 {
		return sdk.WithStack(sdk.ErrNotFound)
	}
	return nil
}

// PostGet load all the groups for an access token
func (a *accessToken) PostGet(db gorp.SqlExecutor) error {
	// Load the user
	u, err := user.LoadUserWithoutAuthByID(db, a.UserID)
	if err != nil {
		return err
	}
	a.User = *u

	// Load the groups
	var groupIDs []int64
	if _, err := db.Select(&groupIDs, "select group_id from access_token_group where access_token_id = $1", a.ID); err != nil {
		return sdk.WrapError(err, "unable to load group id for token %s", a.ID)
	}

	for _, groupID := range groupIDs {
		g, err := group.LoadGroupByID(db, groupID)
		if err != nil {
			log.Error("accessToken.PostGet> unable to load group %d for token %s: %v", groupID, a.ID, err)
			continue
		}
		a.Groups = append(a.Groups, *g)
	}

	return nil
}

// PostUpdate updates relation between access_token and group
func (a *accessToken) PostUpdate(db gorp.SqlExecutor) error {
	return a.PostInsert(db)
}

// PostInsert inserts relation between access_token and group
func (a *accessToken) PostInsert(db gorp.SqlExecutor) error {
	groupIDs := sdk.GroupsToIDs(a.Groups)
	// Insert all groupIDs at one using unnest : https://www.postgresql.org/docs/9.2/functions-array.html
	// UNNEST expand an array to a set of rows.
	// The named parameter must be explicitly casted as an Array (::BIGINT[])
	// the PQ lib is able to scan/value an array with the function pq.Array
	query := "INSERT INTO access_token_group (access_token_id, group_id) (SELECT $1, UNNEST($2::BIGINT[])) ON CONFLICT DO NOTHING"
	if _, err := db.Exec(query, a.ID, pq.Array(groupIDs)); err != nil {
		return sdk.WrapError(err, "unable to insert access_token_group")
	}

	return nil
}
