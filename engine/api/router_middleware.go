package api

import (
	"context"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) authMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	var err error
	var shouldContinue bool
	// Providers (like arsenal)
	ctx, shouldContinue, err = api.authAllowProviderMiddleware(ctx, w, req, rc)
	if err != nil {
		return ctx, sdk.WithStack(err)
	}
	if !shouldContinue {
		return ctx, nil
	}
	// Tokens (like izanamy)
	ctx, shouldContinue, err = api.authStatusTokenMiddleware(ctx, w, req, rc)
	if err != nil {
		return ctx, sdk.WithStack(err)
	}
	if !shouldContinue {
		return ctx, nil
	}

	// JWT base authentification
	ctx, shouldContinue, err = api.authJWTMiddleware(ctx, w, req, rc)
	if err != nil {
		log.Warning("api.router> authentication failed with JWT token: %v", err)
		return ctx, sdk.WithStack(err)
	}
	if !shouldContinue {
		log.Info("api.router> authentication successful with JWT token")
		return ctx, nil
	} else {
		log.Debug("api.router> authentication unsuccessful with JWT token")
	}

	//Check Authentication (users, workers, hatcheries, services)
	ctx, shouldContinue, err = api.authDeprecatedMiddleware(ctx, w, req, rc)
	if err != nil {
		return ctx, sdk.WithStack(err)
	}
	if !shouldContinue {
		return ctx, nil
	}

	return ctx, nil
}

func (api *API) deletePermissionMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	if req.Method == "POST" || req.Method == "PUT" || req.Method == "DELETE" {
		api.deleteUserPermissionCache(ctx, api.Cache)
	}
	return ctx, nil
}

func (api *API) tracingMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	return TracingMiddlewareFunc(api.ServiceName, api.mustDB(), api.Cache)(ctx, w, req, rc)
}

func TracingPostMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	ctx, err := observability.End(ctx, w, req)
	return ctx, err
}

func TracingMiddlewareFunc(serviceName string, db gorp.SqlExecutor, store cache.Store) service.Middleware {
	return func(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
		name := runtime.FuncForPC(reflect.ValueOf(rc.Handler).Pointer()).Name()
		name = strings.Replace(name, ".func1", "", 1)

		splittedName := strings.Split(name, ".")
		name = splittedName[len(splittedName)-1]

		opts := observability.Options{
			Name:     name,
			Enable:   rc.Options["trace_enable"] == "true",
			Init:     rc.Options["trace_new_trace"] == "true",
			User:     deprecatedGetUser(ctx),
			Worker:   getWorker(ctx),
			Hatchery: getHatchery(ctx),
		}

		ctx, err := observability.Start(ctx, serviceName, w, req, opts, db, store)
		newReq := req.WithContext(ctx)
		*req = *newReq

		return ctx, err
	}
}

func (api *API) maintenanceMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	if rc.Options["maintenance_aware"] == "true" && api.Maintenance {
		return ctx, sdk.WrapError(sdk.ErrServiceUnavailable, "CDS Maintenance ON")
	}
	return ctx, nil
}
