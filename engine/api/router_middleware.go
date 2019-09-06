package api

import (
	"context"
	"net/http"

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

func (api *API) maintenanceMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	if rc.Options["maintenance_aware"] == "true" && api.Maintenance {
		return ctx, sdk.WrapError(sdk.ErrServiceUnavailable, "CDS Maintenance ON")
	}
	return ctx, nil
}
