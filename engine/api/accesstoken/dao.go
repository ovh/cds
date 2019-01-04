package accesstoken

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
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
	if err := db.Insert(&token); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// Update a token in database
func Update(db gorp.SqlExecutor, token *sdk.AccessToken) error {
	n, err := db.Update(&token)
	if err != nil {
		return sdk.WithStack(err)
	}
	if n < 1 {
		return sdk.WithStack(sdk.ErrNotFound)
	}
	return nil
}

// Delete a token in database
func Delete(db gorp.SqlExecutor, token *sdk.AccessToken) error {
	n, err := db.Delete(token)
	if err != nil {
		return sdk.WithStack(err)
	}
	if n < 1 {
		return sdk.WithStack(sdk.ErrNotFound)
	}
	return nil
}

func (a *accessToken) PostGet(s gorp.SqlExecutor) error {
	return nil
}

func (a *accessToken) PostUpdate(s gorp.SqlExecutor) error {
	return nil
}

func (a *accessToken) PostInsert(s gorp.SqlExecutor) error {
	return nil
}
