package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Old and Deprecated Authentication
func (api *API) authDeprecatedMiddleware_DEPRECATED(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, bool, error) {
	headers := req.Header

	if rc.Options["auth"] != "true" {
		return ctx, false, nil
	}

	if rc.Options["auth"] == "true" && getProvider(ctx) == nil {
		switch headers.Get("User-Agent") {
		case sdk.WorkerAgent:
			log.Debug("authDeprecatedMiddleware.WorkerAgent")
			var err error
			ctx, err = auth.CheckWorkerAuth(ctx, api.mustDB(), api.Cache, headers)
			if err != nil {
				return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s sdk.WorkerAgent agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		case sdk.ServiceAgent:
			log.Debug("authDeprecatedMiddleware.ServiceAgent")
			var err error
			ctx, err = auth.CheckServiceAuth(ctx, api.mustDB(), api.Cache, headers)
			if err != nil {
				return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s sdk.ServiceAgent agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		default:
			log.Debug("authDeprecatedMiddleware.CheckAuth_DEPRECATED")
			var err error
			ctx, err = auth.CheckAuth_DEPRECATED(ctx, w, req, api.mustDB())
			if err != nil {
				return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		}
	}

	if deprecatedGetUser(ctx) == nil {
		return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Unable to find connected user")
	}

	if rc.Options["allowServices"] == "true" && getService(ctx) != nil {
		return ctx, false, nil
	}

	if rc.Options["needHatchery"] == "true" && getHatchery(ctx) != nil {
		return ctx, false, nil
	}

	if rc.Options["needService"] == "true" {
		if getService(ctx) != nil {
			return ctx, false, nil
		}
		return ctx, false, sdk.WrapError(sdk.ErrForbidden, "Router> Need service")
	}

	if rc.Options["needWorker"] == "true" {
		permissionOk := api.checkWorkerPermission(ctx, api.mustDB(), rc, mux.Vars(req))
		if !permissionOk {
			return ctx, false, sdk.WrapError(sdk.ErrForbidden, "Router> Worker not authorized")
		}
		return ctx, false, nil
	}

	if deprecatedGetUser(ctx).Admin {
		return ctx, false, nil
	}

	if rc.Options["needAdmin"] != "true" {
		if err := api.checkPermission(ctx, mux.Vars(req), getPermissionByMethod(req.Method, rc.Options["isExecution"] == "true")); err != nil {
			return ctx, false, err
		}
	} else {
		return ctx, false, sdk.WrapError(sdk.ErrForbidden, "Router> User not authorized (needAdmin)")
	}

	if rc.Options["needUsernameOrAdmin"] == "true" && deprecatedGetUser(ctx).Username != mux.Vars(req)["username"] {
		// get / update / delete user -> for admin or current user
		// if not admin and currentUser != username in request -> ko
		return ctx, false, sdk.WrapError(sdk.ErrForbidden, "Router> User not authorized on this resource")
	}

	return ctx, true, nil
}
