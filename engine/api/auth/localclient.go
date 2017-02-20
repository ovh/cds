package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//LocalClient is a auth driver wich store all in database
type LocalClient struct {
	store sessionstore.Store
}

//Open nothing
func (c *LocalClient) Open(options interface{}, store sessionstore.Store) error {
	log.Notice("Auth> Connecting to session store")
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

const (
	//LocalClientSessionMode for SessionToken auth only
	LocalClientSessionMode = iota
	//LocalClientBasicAuthMode for Basic auth only
	LocalClientBasicAuthMode
)

//GetCheckAuthHeaderFunc returns the func to heck http headers.
//Options is a const to switch from session to basic auth or both
func (c *LocalClient) GetCheckAuthHeaderFunc(options interface{}) func(db *gorp.DbMap, headers http.Header, ctx *context.Ctx) error {
	switch options {
	case LocalClientBasicAuthMode:
		return func(db *gorp.DbMap, headers http.Header, ctx *context.Ctx) error {
			if h := headers.Get(sdk.AuthHeader); h != "" {
				if err := checkWorkerAuth(db, h, ctx); err != nil {
					return err
				}
				return nil
			}

			//Even if we are in BasicAuthMode, we may receive session token (from template extension by example)
			sessionToken := headers.Get(sdk.SessionTokenHeader)
			if sessionToken != "" {
				return c.checkUserSessionAuth(db, headers, ctx)
			}

			//Standard way : basic auth
			h := headers.Get("Authorization")
			if h == "" {
				return fmt.Errorf("no authorization header")
			}
			return c.checkUserBasicAuth(db, h, ctx)
		}
	case LocalClientSessionMode:
		return func(db *gorp.DbMap, headers http.Header, ctx *context.Ctx) error {
			//Check if its a worker
			if h := headers.Get(sdk.AuthHeader); h != "" {
				if err := checkWorkerAuth(db, h, ctx); err != nil {
					return err
				}
				return nil
			}
			//Check if its comming from CLI
			if headers.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
				if getUserPersistentSession(db, c.Store(), headers, ctx) {
					return nil
				}
				if reloadUserPersistentSession(db, c.Store(), headers, ctx) {
					return nil
				}
			}

			return c.checkUserSessionAuth(db, headers, ctx)
		}
	default:
		return func(db *gorp.DbMap, headers http.Header, c *context.Ctx) error {
			return fmt.Errorf("invalid authorization mechanism")
		}
	}
}

func (c *LocalClient) checkUserBasicAuth(db gorp.SqlExecutor, authHeaderValue string, ctx *context.Ctx) error {
	// Split Basic and (user:pass)64
	auth := strings.SplitN(authHeaderValue, " ", 2)
	if len(auth) != 2 || auth[0] != "Basic" {
		return fmt.Errorf("bad authorization header syntax")
	}

	userPwd, _ := base64.StdEncoding.DecodeString(auth[1])
	userPwdArray := strings.SplitN(string(userPwd), ":", 2)
	if len(userPwdArray) != 2 {
		return fmt.Errorf("bad authorization header syntax")
	}

	// Load user
	u, err := user.LoadUserAndAuth(db, userPwdArray[0])
	if err != nil {
		return err
	}

	// Verify password
	loginOk, err := c.AuthentifyUser(db, u, userPwdArray[1])
	if err != nil {
		return err
	}
	if !loginOk {
		return fmt.Errorf("bad password")
	}
	ctx.User = u
	return nil
}

func (c *LocalClient) checkUserSessionAuth(db gorp.SqlExecutor, headers http.Header, ctx *context.Ctx) error {
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
