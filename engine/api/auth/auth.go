package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/user"
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
	ContextProvider
	ContextUserAuthentified
	ContextGrantedUser
	ContextJWT
	ContextScope
)

var (
	Store sessionstore.Store
)

//NewSession inits a new session
func NewSession(u *sdk.AuthentifiedUser) (sessionstore.SessionKey, error) {
	session, err := Store.New("")
	if err != nil {
		return "", err
	}
	log.Info("Auth> New Session for %s", u.Username)
	Store.Set(session, "username", u.Username)
	return session, err
}

//GetUsername retrieve the username from the token
func GetUsername(token string) (string, error) {
	var username string
	err := Store.Get(sessionstore.SessionKey(token), "username", &username)
	if err != nil {
		return "", err
	}
	if username == "" {
		return "", nil
	}
	return username, nil
}

//GetWorker returns the worker instance from its id
func GetWorker(db *gorp.DbMap, Store cache.Store, workerID, workerName string) (*sdk.Worker, error) {
	// Load worker
	var w = &sdk.Worker{}

	key := cache.Key("worker", workerID)
	b := Store.Get(key, w)
	if !b || w.ActionBuildID == 0 {
		var err error
		w, err = worker.LoadWorker(db, workerID)
		if err != nil {
			return nil, fmt.Errorf("cannot load worker '%s': %s", workerName, err)
		}
		Store.Set(key, w)
	}
	return w, nil
}

//GetService returns the service instance from its hash
func GetService(db *gorp.DbMap, Store cache.Store, hash string) (*sdk.Service, error) {
	//Load the service from the cache
	//TODO: this should be embeded in the repository layer
	var srv = &sdk.Service{}
	key := cache.Key("services", hash)
	// Else load it from DB
	if !Store.Get(key, srv) {
		var err error
		srv, err = services.FindByHash(db, hash)
		if err != nil {
			return nil, fmt.Errorf("cannot load service: %s", err)
		}
		if srv.GroupID != nil && group.SharedInfraGroup.ID == *srv.GroupID {
			srv.IsSharedInfra = true
			srv.Uptodate = srv.Version == sdk.VERSION
		}
		Store.Set(key, srv)
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
func CheckWorkerAuth(ctx context.Context, db *gorp.DbMap, Store cache.Store, headers http.Header) (context.Context, error) {
	id, err := base64.StdEncoding.DecodeString(headers.Get(sdk.AuthHeader))
	if err != nil {
		return ctx, fmt.Errorf("bad worker key syntax: %s", err)
	}
	workerID := string(id)

	name := headers.Get(cdsclient.RequestedNameHeader)
	w, err := GetWorker(db, Store, workerID, name)
	if err != nil {
		return ctx, err
	}

	//TODO
	// Worker authentication against jwt token

	ctx = context.WithValue(ctx, ContextUser, &sdk.User{Username: w.Name})
	ctx = context.WithValue(ctx, ContextWorker, w)
	return ctx, nil
}

// CheckServiceAuth checks services authentication
func CheckServiceAuth(ctx context.Context, db *gorp.DbMap, Store cache.Store, headers http.Header) (context.Context, error) {
	id, err := base64.StdEncoding.DecodeString(headers.Get(sdk.AuthHeader))
	if err != nil {
		return ctx, fmt.Errorf("bad service key syntax: %s", err)
	}

	serviceHash := string(id)
	if serviceHash == "" {
		return ctx, fmt.Errorf("missing service Hash")
	}

	srv, err := GetService(db, Store, serviceHash)
	if err != nil {
		return ctx, err
	}
	//TODO
	// Service authentication against jwt token

	ctx = context.WithValue(ctx, ContextUser, &sdk.User{Username: srv.Name})
	if srv.Type == services.TypeHatchery {
		ctx = context.WithValue(ctx, ContextHatchery, srv)
	} else {
		ctx = context.WithValue(ctx, ContextService, srv)
	}
	return ctx, nil
}

// GetEphemeralSession_DEPRECATED have to be deprecated
func GetEphemeralSession_DEPRECATED(ctx context.Context, db gorp.SqlExecutor, sessionToken, username string) (context.Context, error) {
	u, err := user.LoadUserByUsername(db, username)
	if err != nil {
		return ctx, err
	}

	oldUser, err := user.GetDeprecatedUser(db, u)
	if err != nil {
		return ctx, err
	}

	ctx = context.WithValue(ctx, ContextUser, oldUser)
	ctx = context.WithValue(ctx, ContextUserAuthentified, u)
	return ctx, nil
}

//CheckAuth checks the auth
func CheckAuth_DEPRECATED(ctx context.Context, w http.ResponseWriter, req *http.Request, db gorp.SqlExecutor) (context.Context, error) {
	//Check persistent session
	if req.Header.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
		var ok bool
		ctx, ok = getUserPersistentSession_DEPRECATED(ctx, db, req.Header)
		if ok {
			return ctx, nil
		}
		return ctx, sdk.WithStack(sdk.ErrSessionNotFound)
	}

	//Check other session
	sessionToken := req.Header.Get(sdk.SessionTokenHeader)
	if sessionToken == "" {
		//Accept session in request
		sessionToken = req.FormValue("session")
	}
	if sessionToken == "" {
		return ctx, sdk.WithStack(sdk.ErrSessionNotFound)
	}

	exists, err := Store.Exists(sessionstore.SessionKey(sessionToken))
	if err != nil {
		return ctx, sdk.WithStack(err)
	}

	username, err := GetUsername(sessionToken)
	if err != nil {
		return ctx, sdk.WithStack(err)
	}

	ctx, err = GetEphemeralSession_DEPRECATED(ctx, db, sessionToken, username)
	if err != nil {
		return ctx, sdk.WithStack(fmt.Errorf("authorization failed for %s: %v", username, err))
	}

	if !exists {
		return ctx, sdk.WithStack(sdk.ErrSessionNotFound)
	}

	return ctx, nil
}
