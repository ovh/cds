package accesstoken

import (
	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type accessToken sdk.AccessToken

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

type accessTokenGroup struct {
	AccessTokenID string `db:"access_token_id"`
	GroupID       int64  `db:"group_id"`
	// aggregates
	Group *sdk.Group `db:"-"`
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(accessToken{}, "access_token", false, "id"),
		gorpmapping.New(accessTokenGroup{}, "access_token_group", false, "access_token_id", "group_id"),
	)
}
