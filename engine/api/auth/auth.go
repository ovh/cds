package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

type contextKey int

const (
	ContextUser contextKey = iota
	ContextHatchery
	ContextWorker
	ContextService
	ContextUserSession
)

//Driver is an interface to all auth method (local, ldap and beyond...)
type Driver interface {
	Open(options interface{}, store sessionstore.Store) error
	Store() sessionstore.Store
	CheckAuth(ctx context.Context, w http.ResponseWriter, req *http.Request) (context.Context, error)
	Authentify(username, password string) (bool, error)
}

//GetDriver is a factory
func GetDriver(c context.Context, mode string, options interface{}, storeOptions sessionstore.Options, DBFunc func() *gorp.DbMap) (Driver, error) {
	log.Info("Auth> Initializing driver (%s)", mode)
	store, err := sessionstore.Get(c, storeOptions.Cache, storeOptions.TTL)
	if err != nil {
		return nil, fmt.Errorf("unable to get AuthDriver : %v", err)
	}

	var d Driver
	switch mode {
	case "ldap":
		d = &LDAPClient{
			dbFunc: DBFunc,
		}
	default:
		d = &LocalClient{
			dbFunc: DBFunc,
		}
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

//GetWorker returns the worker instance from its id
func GetWorker(db *gorp.DbMap, store cache.Store, workerID, workerName string) (*sdk.Worker, error) {
	// Load worker
	var w = &sdk.Worker{}

	key := cache.Key("worker", workerID)
	// Else load it from DB
	if !store.Get(key, w) {
		var err error
		w, err = worker.LoadWorker(db, workerID)
		if err != nil {
			return nil, fmt.Errorf("cannot load worker '%s': %s", workerName, err)
		}
		store.Set(key, w)
	}

	return w, nil
}

//GetService returns the service instance from its hash
func GetService(db *gorp.DbMap, store cache.Store, hash string) (*sdk.Service, error) {
	//Load the service from the cache
	//TODO: this should be embeded in the repository layer
	var srv = &sdk.Service{}
	key := cache.Key("services", hash)
	// Else load it from DB
	if !store.Get(key, srv) {
		var err error
		repo := services.NewRepository(func() *gorp.DbMap { return db }, store)
		srv, err = repo.FindByHash(hash)
		if err != nil {
			return nil, fmt.Errorf("cannot load service: %s", err)
		}
		store.Set(key, srv)
	}

	return srv, nil
}

// ContextValues retuns auth values of a context
func ContextValues(ctx context.Context) map[interface{}]interface{} {
	return map[interface{}]interface{}{
		ContextHatchery: ctx.Value(ContextHatchery),
		ContextService:  ctx.Value(ContextService),
		ContextWorker:   ctx.Value(ContextWorker),
		ContextUser:     ctx.Value(ContextUser),
	}
}

// CheckWorkerAuth checks worker authentication
func CheckWorkerAuth(ctx context.Context, db *gorp.DbMap, store cache.Store, headers http.Header) (context.Context, error) {
	id, err := base64.StdEncoding.DecodeString(headers.Get(sdk.AuthHeader))
	if err != nil {
		return ctx, fmt.Errorf("bad worker key syntax: %s", err)
	}
	workerID := string(id)

	name := headers.Get(cdsclient.RequestedNameHeader)
	w, err := GetWorker(db, store, workerID, name)
	if err != nil {
		return ctx, err
	}

	ctx = context.WithValue(ctx, ContextUser, &sdk.User{Username: w.Name})
	ctx = context.WithValue(ctx, ContextWorker, w)
	return ctx, nil
}

// CheckServiceAuth checks services authentication
func CheckServiceAuth(ctx context.Context, db *gorp.DbMap, store cache.Store, headers http.Header) (context.Context, error) {
	id, err := base64.StdEncoding.DecodeString(headers.Get(sdk.AuthHeader))
	if err != nil {
		return ctx, fmt.Errorf("bad service key syntax: %s", err)
	}

	serviceHash := string(id)

	srv, err := GetService(db, store, serviceHash)
	if err != nil {
		return ctx, err
	}

	ctx = context.WithValue(ctx, ContextUser, &sdk.User{Username: srv.Name})
	ctx = context.WithValue(ctx, ContextService, srv)
	return ctx, nil
}

// CheckHatcheryAuth checks hatchery authentication
func CheckHatcheryAuth(ctx context.Context, db *gorp.DbMap, headers http.Header) (context.Context, error) {
	uid, err := base64.StdEncoding.DecodeString(headers.Get(sdk.AuthHeader))
	if err != nil {
		return ctx, fmt.Errorf("bad worker key syntax: %s", err)
	}

	name := headers.Get(cdsclient.RequestedNameHeader)
	h, err := hatchery.LoadHatchery(db, string(uid), name)
	if err != nil {
		return ctx, fmt.Errorf("Invalid Hatchery UID:%s name:%s err:%s", string(uid), name, err)
	}

	ctx = context.WithValue(ctx, ContextUser, &sdk.User{Username: h.Name})
	ctx = context.WithValue(ctx, ContextHatchery, h)
	return ctx, nil
}
