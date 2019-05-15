package user

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/sdk"
)

// NewPersistentSession_DEPRECATED creates a new persistent session token in database
func NewPersistentSession_DEPRECATED(db gorp.SqlExecutor, u *sdk.User) (sessionstore.SessionKey, error) {
	t, errSession := sessionstore.NewSessionKey()
	if errSession != nil {
		return "", errSession
	}
	newToken := sdk.UserToken{
		Token:              string(t),
		Comment:            fmt.Sprintf("New persistent session for %s", u.Username),
		CreationDate:       time.Now(),
		LastConnectionDate: time.Now(),
		UserID:             u.ID,
	}

	if err := InsertPersistentSessionToken(db, newToken); err != nil {
		return "", err
	}
	return t, nil
}

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
