package user

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
)

// LoadPersistentSessionToken load a token from the database
func LoadPersistentSessionToken(db gorp.SqlExecutor, k sessionstore.SessionKey) (*sdk.UserToken, error) {
	tdb := persistentSessionToken{}
	if err := db.SelectOne(&tdb, "select * from user_persistent_session where token = $1", string(k)); err != nil {
		return nil, err
	}
	t := sdk.UserToken(tdb)
	return &t, nil
}

// InsertPersistentSessionToken create a new persistent session
func InsertPersistentSessionToken(db gorp.SqlExecutor, t sdk.UserToken) error {
	tdb := persistentSessionToken(t)
	if err := db.Insert(&tdb); err != nil {
		return sdk.WrapError(err, "Unable to insert persistent session token for user %d", t.UserID)
	}
	return nil
}

// UpdatePersistentSessionToken updates a persistent session
func UpdatePersistentSessionToken(db gorp.SqlExecutor, t sdk.UserToken) error {
	tdb := persistentSessionToken(t)
	if _, err := db.Update(&tdb); err != nil {
		return sdk.WrapError(err, "Unable to update persistent session token for user %d", t.UserID)
	}
	return nil
}
