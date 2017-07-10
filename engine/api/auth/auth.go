package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
		if getUserPersistentSession(db, store, headers, ctx) {
			return true
		}
		if reloadUserPersistentSession(db, store, headers, ctx) {
			return true
		}
	}
	return false
}

func getUserPersistentSession(db gorp.SqlExecutor, store sessionstore.Store, headers http.Header, ctx *ctx.Ctx) bool {
	h := headers.Get(sdk.SessionTokenHeader)
	if h != "" {
		ok, _ := store.Exists(sessionstore.SessionKey(h))
		if ok {
			var usr string
			store.Get(sessionstore.SessionKey(h), "username", &usr)
			//Set user in ctx
			u, err := user.LoadUserWithoutAuth(db, usr)
			if err != nil {
				log.Warning("getUserPersistentSession> Unable to load user %s", usr)
				return false
			}
			ctx.User = u
			return true
		}
	}
	return false
}

func reloadUserPersistentSession(db gorp.SqlExecutor, store sessionstore.Store, headers http.Header, ctx *ctx.Ctx) bool {
	authHeaderValue := headers.Get("Authorization")
	if authHeaderValue == "" {
		log.Warning("ReloadUserPersistentSession> No Authorization Header")
		return false
	}
	// Split Basic and (user:pass)64
	auth := strings.SplitN(authHeaderValue, " ", 2)
	if len(auth) != 2 || auth[0] != "Basic" {
		log.Warning("ReloadUserPersistentSession> Wrong Authorization header syntax")
		return false
	}

	userPwd, _ := base64.StdEncoding.DecodeString(auth[1])
	userPwdArray := strings.SplitN(string(userPwd), ":", 2)
	if len(userPwdArray) != 2 {
		log.Warning("ReloadUserPersistentSession> Authorization failed")
		return false
	}

	// Load user
	u, err1 := user.LoadUserAndAuth(db, userPwdArray[0])
	if err1 != nil {
		log.Warning("ReloadUserPersistentSession> Authorization failed")
		return false
	}
	ctx.User = u

	// Verify token
	for _, t := range u.Auth.Tokens {
		if t.Token == userPwdArray[1] {
			log.Debug("ReloadUserPersistentSession> Persistent session successfully reloaded %s %s", u.Username, t.Token)
			if _, err := store.New(sessionstore.SessionKey(t.Token)); err != nil {
				log.Warning("ReloadUserPersistentSession> Unable to create new session %s:%s", t.Token, err)
				return false
			}
			store.Set(sessionstore.SessionKey(t.Token), "username", u.Username)
			return true
		}
	}

	log.Warning("ReloadUserPersistentSession> failed for username:%s", u.Username)
	return false
}

//GetWorker returns the worker instance from its id
func GetWorker(db gorp.SqlExecutor, workerID string) (*sdk.Worker, error) {
	// Load worker
	var w *sdk.Worker
	var oldWorker sdk.Worker

	// Try to load worker from cache
	key := cache.Key("worker", workerID)
	cache.Get(key, &oldWorker)
	var putWorkerInCache bool
	if oldWorker.ID != "" {
		w = &oldWorker
	}

	// Else load it from DB
	if w == nil {
		var err error
		w, err = worker.LoadWorker(db, workerID)
		if err != nil {
			return nil, fmt.Errorf("cannot load worker: %s", err)
		}
		putWorkerInCache = true
	}

	if putWorkerInCache {
		//Set the worker in cache
		cache.Set(key, *w)
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
