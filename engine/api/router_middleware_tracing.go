package api

import (
	"context"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk/telemetry"
)

func (api *API) tracingMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	return TracingMiddlewareFunc(api, api.mustDB(), api.Cache)(ctx, w, req, rc)
}

func TracingPostMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	ctx, err := telemetry.End(ctx, w, req)
	return ctx, err
}

func TracingMiddlewareFunc(s service.Service, db gorp.SqlExecutor, store cache.Store) service.Middleware {
	return func(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
		name := runtime.FuncForPC(reflect.ValueOf(rc.Handler).Pointer()).Name()
		name = strings.Replace(name, ".func1", "", 1)

		splittedName := strings.Split(name, ".")
		name = splittedName[len(splittedName)-1]

		opts := telemetry.Options{
			Name: name,
		}

		ctx, err := observability.Start(ctx, s, w, req, opts, gorpmapping.Mapper, db, store)
		newReq := req.WithContext(ctx)
		*req = *newReq

		return ctx, err
	}
}
