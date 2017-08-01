package auth

import (
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//LocalClient is a auth driver which store all in database
type LocalClient struct {
	store sessionstore.Store
}

//Open nothing
func (c *LocalClient) Open(options interface{}, store sessionstore.Store) error {
	log.Info("Auth> Connecting to session store")
	c.store = store
	return nil
}

//Store returns store
func (c *LocalClient) Store() sessionstore.Store {
	return c.store
}

//Authentify check username and password
func (c *LocalClient) Authentify(db gorp.SqlExecutor, username, password string) (bool, error) {
	// Load user
	u, err := user.LoadUserAndAuth(db, username)
	if err != nil {
		log.Warning("Auth> Authorization failed")
		return false, err
	}
	b := user.IsCheckValid(password, u.Auth.HashedPassword)
	return b, err
}

//AuthentifyUser check password in database
func (c *LocalClient) AuthentifyUser(db gorp.SqlExecutor, u *sdk.User, password string) (bool, error) {
	return user.IsCheckValid(password, u.Auth.HashedPassword), nil
}

//CheckAuthHeader checks http headers.
func (c *LocalClient) CheckAuthHeader(db *gorp.DbMap, headers http.Header, ctx *businesscontext.Ctx) error {
	//Check if its coming from CLI
	if headers.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
		if getUserPersistentSession(db, c.Store(), headers, ctx) {
			return nil
		}
	}
	return c.checkUserSessionAuth(db, headers, ctx)
}

func (c *LocalClient) checkUserSessionAuth(db gorp.SqlExecutor, headers http.Header, ctx *businesscontext.Ctx) error {
	sessionToken := headers.Get(sdk.SessionTokenHeader)
	if sessionToken == "" {
		return fmt.Errorf("no session header")
	}
	exists, err := c.store.Exists(sessionstore.SessionKey(sessionToken))
	if err != nil {
		return err
	}
	username, err := GetUsername(c.store, sessionToken)
	if err != nil {
		return err
	}
	u, err := user.LoadUserAndAuth(db, username)
	if err != nil {
		return fmt.Errorf("authorization failed for %s: %s", username, err)
	}
	ctx.User = u

	if !exists {
		return fmt.Errorf("invalid session")
	}
	return nil
}
