package auth

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//NewPersistentSession create a new session with token stored as user_key in database
func NewPersistentSession(db gorp.SqlExecutor, u *sdk.AuthentifiedUser) (sessionstore.SessionKey, error) {
	u, err := user.LoadUserByUsername(db, u.Username)
	if err != nil {
		return "", err
	}

	oldUser, err := user.GetDeprecatedUser(db, u)
	if err != nil {
		return "", err
	}

	t, errSession := user.NewPersistentSession_DEPRECATED(db, oldUser)
	if errSession != nil {
		return "", errSession
	}

	session, errStore := Store.New(t)
	if errStore != nil {
		return "", errStore
	}
	log.Info("NewPersistentSession> New Session for %s", u.Username)
	Store.Set(session, "username", u.Username)

	return session, nil
}

func getUserPersistentSession_DEPRECATED(ctx context.Context, db gorp.SqlExecutor, headers http.Header) (context.Context, bool) {
	h := headers.Get(sdk.SessionTokenHeader)
	if h == "" {
		return ctx, false
	}

	key := sessionstore.SessionKey(h)
	ok, _ := Store.Exists(key)
	var err error
	var u *sdk.AuthentifiedUser

	if !ok {
		//Reload the persistent session from the database
		token, err := user.LoadPersistentSessionToken(db, key)
		if err != nil {
			log.Warning("getUserPersistentSession> Unable to load user by token %s (%v)", key, err)
			return ctx, false
		}
		u, err = user.LoadByOldUserID(db, token.UserID)
		if err == nil {
			Store.New(key)
			Store.Set(key, "username", u.Username)

		}
	} else {
		//The session is in the session store
		var usr string
		Store.Get(sessionstore.SessionKey(h), "username", &usr)
		u, err = user.LoadUserByUsername(db, usr)
	}

	//Check previous errors
	if err != nil {
		log.Warning("getUserPersistentSession> Unable to load user")
		return ctx, false
	}

	oldUser, err := user.GetDeprecatedUser(db, u)
	if err != nil {
		return ctx, false
	}

	//Set user in ctx
	ctx = context.WithValue(ctx, ContextUser, oldUser)
	ctx = context.WithValue(ctx, ContextUserAuthentified, u)

	//Launch update of the persistent session token
	token, err := user.LoadPersistentSessionToken(db, key)
	if err != nil {
		log.Warning("getUserPersistentSession> Unable to load user by token %s: %v", key, err)
		if sdk.Cause(err) == sql.ErrNoRows {
			if err := Store.Delete(key); err != nil {
				log.Error("getUserPersistentSession> Unable to delete session %v", key)
			}
		}
		return ctx, false
	}
	token.LastConnectionDate = time.Now()
	if err := user.UpdatePersistentSessionToken(db, *token); err != nil {
		log.Error("getUserPersistentSession> Unable to update token")
		if err := Store.Delete(key); err != nil {
			log.Error("getUserPersistentSession> Unable to delete session %v", key)
		}
		return ctx, false
	}

	return ctx, true
}
