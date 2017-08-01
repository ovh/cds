package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"

	ctx "github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Driver is an interface to all auth method (local, ldap and beyond...)
type Driver interface {
	Open(options interface{}, store sessionstore.Store) error
	Store() sessionstore.Store
	Authentify(db gorp.SqlExecutor, username, password string) (bool, error)
	AuthentifyUser(db gorp.SqlExecutor, user *sdk.User, password string) (bool, error)
	CheckAuthHeader(db *gorp.DbMap, headers http.Header, c *ctx.Ctx) error
}

//GetDriver is a factory
func GetDriver(c context.Context, mode string, options interface{}, storeOptions sessionstore.Options) (Driver, error) {
	log.Info("Auth> Intializing driver (%s)", mode)
	store, err := sessionstore.Get(c, storeOptions.Mode, storeOptions.RedisHost, storeOptions.RedisPassword, storeOptions.TTL)
	if err != nil {
		return nil, fmt.Errorf("unable to get AuthDriver : %v", err)
	}

	var d Driver
	switch mode {
	case "ldap":
		d = &LDAPClient{}
	default:
		d = &LocalClient{}
	}

	if d == nil {
		return nil, errors.New("GetDriver> Unable to get AuthDriver (nil)")
	}
	if err := d.Open(options, store); err != nil {
		return nil, sdk.WrapError(err, "GetDriver> Unable to get AuthDriver")
	}
	return d, nil
}

//NewSession inits a new session
func NewSession(d Driver, u *sdk.User) (sessionstore.SessionKey, error) {
	session, err := d.Store().New("")
	if err != nil {
		return "", err
	}
	log.Info("Auth> New Session for %s", u.Username)
	d.Store().Set(session, "username", u.Username)
	return session, err
}

//NewPersistentSession create a new session with token stored as user_key in database
func NewPersistentSession(db gorp.SqlExecutor, d Driver, u *sdk.User) (sessionstore.SessionKey, error) {
	u, errLoad := user.LoadUserAndAuth(db, u.Username)
	if errLoad != nil {
		return "", errLoad
	}

	t, errSession := user.NewPersistentSession(db, u)
	if errSession != nil {
		return "", errSession
	}

	session, errStore := d.Store().New(t)
	if errStore != nil {
		return "", errStore
	}
	log.Info("NewPersistentSession> New Session for %s", u.Username)
	d.Store().Set(session, "username", u.Username)

	return session, nil
}

//GetUsername retrieve the username from the token
func GetUsername(store sessionstore.Store, token string) (string, error) {
	var username string
	err := store.Get(sessionstore.SessionKey(token), "username", &username)
	if err != nil {
		return "", err
	}
	if username == "" {
		return "", nil
	}
	return username, nil
}

//CheckPersistentSession check persistent session token from CLI
func CheckPersistentSession(db gorp.SqlExecutor, store sessionstore.Store, headers http.Header, ctx *ctx.Ctx) bool {
	if headers.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
		return getUserPersistentSession(db, store, headers, ctx)
	}
	return false
}

func getUserPersistentSession(db gorp.SqlExecutor, store sessionstore.Store, headers http.Header, ctx *ctx.Ctx) bool {
	h := headers.Get(sdk.SessionTokenHeader)
	if h == "" {
		return false
	}

	key := sessionstore.SessionKey(h)
	ok, _ := store.Exists(key)
	var err error
	var u *sdk.User

	if !ok {
		//Reload the persistent session from the database
		token, err := user.LoadPersistentSessionToken(db, key)
		if err != nil {
			log.Warning("getUserPersistentSession> Unable to load user by token %s", key)
			return false
		}
		u, err = user.LoadUserWithoutAuthByID(db, token.UserID)
		store.New(key)
		store.Set(key, "username", u.Username)

	} else {
		//The session is in the session store
		var usr string
		store.Get(sessionstore.SessionKey(h), "username", &usr)
		u, err = user.LoadUserWithoutAuth(db, usr)
	}

	//Check previous errors
	if err != nil {
		log.Warning("getUserPersistentSession> Unable to load user")
		return false
	}

	//Set user in ctx
	ctx.User = u

	//Launch update of the persistent session token in background
	defer func(key sessionstore.SessionKey) {
		token, err := user.LoadPersistentSessionToken(db, key)
		if err != nil {
			log.Warning("getUserPersistentSession> Unable to load user by token %s: %v", key, err)
			return
		}
		token.LastConnectionDate = time.Now()
		if err := user.UpdatePersistentSessionToken(db, *token); err != nil {
			log.Error("getUserPersistentSession> Unable to update token")
		}
	}(key)

	return true
}

//GetWorker returns the worker instance from its id
func GetWorker(db gorp.SqlExecutor, workerID string) (*sdk.Worker, error) {
	// Load worker
	var w *sdk.Worker

	key := cache.Key("worker", workerID)
	// Else load it from DB
	if !cache.Get(key, w) {
		var err error
		w, err = worker.LoadWorker(db, workerID)
		if err != nil {
			return nil, fmt.Errorf("cannot load worker: %s", err)
		}
		cache.Set(key, w)
	}

	return w, nil
}

// CheckWorkerAuth checks worker authentication
func CheckWorkerAuth(db *gorp.DbMap, headers http.Header, ctx *ctx.Ctx) error {
	id, err := base64.StdEncoding.DecodeString(headers.Get(sdk.AuthHeader))
	if err != nil {
		return fmt.Errorf("bad worker key syntax: %s", err)
	}
	workerID := string(id)

	w, err := GetWorker(db, workerID)
	if err != nil {
		return err
	}
	ctx.User = &sdk.User{Username: w.Name}
	ctx.Worker = w

	return nil
}

// CheckHatcheryAuth checks hatchery authentication
func CheckHatcheryAuth(db *gorp.DbMap, headers http.Header, c *ctx.Ctx) error {
	uid, err := base64.StdEncoding.DecodeString(headers.Get(sdk.AuthHeader))
	if err != nil {
		return fmt.Errorf("bad worker key syntax: %s", err)
	}

	h, err := hatchery.LoadHatchery(db, string(uid))
	if err != nil {
		return fmt.Errorf("Invalid Hatchery UID:%s err:%s", string(uid), err)
	}

	c.User = &sdk.User{Username: h.Name}
	c.Hatchery = h
	return nil
}
