package api

import (
	"context"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/tracing"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func (api *API) authMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error) {
	headers := req.Header

	// Check Provider
	if rc.Options["allowProvider"] == "true" {
		providerName := req.Header.Get("X-Provider-Name")
		providerToken := req.Header.Get("X-Provider-Token")
		var providerOK bool
		for _, p := range api.Config.Providers {
			if p.Name == providerName && p.Token == providerToken {
				providerOK = true
				break
			}
		}
		if providerOK {
			ctx = context.WithValue(ctx, auth.ContextUser, &sdk.User{Username: providerName, Admin: true})
			ctx = context.WithValue(ctx, auth.ContextProvider, providerName)
			return ctx, nil
		}
	}

	// Check Token
	if h, ok := rc.Options["token"]; ok {
		headerSplitted := strings.Split(h, ":")
		receivedValue := req.Header.Get(headerSplitted[0])
		if receivedValue != headerSplitted[1] {
			return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s", req.Method, req.URL, req.RemoteAddr)
		}
	}

	//Check Authentication
	if rc.Options["auth"] == "true" && getProvider(ctx) == nil {
		switch headers.Get("User-Agent") {
		case sdk.HatcheryAgent:
			var err error
			ctx, err = auth.CheckHatcheryAuth(ctx, api.mustDB(), headers)
			if err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		case sdk.WorkerAgent:
			var err error
			ctx, err = auth.CheckWorkerAuth(ctx, api.mustDB(), api.Cache, headers)
			if err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		case sdk.ServiceAgent:
			var err error
			ctx, err = auth.CheckServiceAuth(ctx, api.mustDB(), api.Cache, headers)
			if err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		default:
			var err error
			ctx, err = api.Router.AuthDriver.CheckAuth(ctx, w, req)
			if err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		}
	}

	//Get the permission for either the hatchery, the worker or the user
	switch {
	case getProvider(ctx) != nil:
	case getHatchery(ctx) != nil:
		g, perm, err := loadPermissionsByGroupID(api.mustDB(), api.Cache, getHatchery(ctx).GroupID)
		if err != nil {
			return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> cannot load group permissions for GroupID %d err:%s", getHatchery(ctx).GroupID, err)
		}
		getUser(ctx).Permissions = perm
		getUser(ctx).Groups = append(getUser(ctx).Groups, g)

	case getWorker(ctx) != nil:
		//Refresh the worker
		workerCtx := getWorker(ctx)
		if err := worker.RefreshWorker(api.mustDB(), workerCtx); err != nil {
			return ctx, sdk.WrapError(err, "Router> Unable to refresh worker")
		}

		if workerCtx.ModelID != 0 {
			// worker have a model, load model, then load model's group
			m, err := worker.LoadWorkerModelByID(api.mustDB(), workerCtx.ModelID)
			if err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> cannot load worker: %s - name:%s modelID:%d", err, workerCtx.Name, workerCtx.ModelID)
			}

			if m.GroupID == group.SharedInfraGroup.ID {
				// it's a shared.infra model, load group from token only: workerCtx.GroupID
				if err := api.setGroupsAndPermissionsFromGroupID(ctx, workerCtx.GroupID); err != nil {
					return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> model shared.infra:%d", m.GroupID)
				}
			} else {
				// this model is not attached to shared.infra group, load group with m.GroupID
				if err := api.setGroupsAndPermissionsFromGroupID(ctx, m.GroupID); err != nil {
					return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> model not shared.infra:%d", m.GroupID)
				}
			}
		} else {
			// worker does not have a model, take group from token only
			if err := api.setGroupsAndPermissionsFromGroupID(ctx, workerCtx.GroupID); err != nil {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> no model, worker not shared.infra:%d", workerCtx.GroupID)
			}
		}
	case getUser(ctx) != nil:
		if err := loadUserPermissions(api.mustDB(), api.Cache, getUser(ctx)); err != nil {
			return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Unable to load user %s permission: %s", getUser(ctx).ID, err)
		}
	}

	if rc.Options["auth"] != "true" {
		return ctx, nil
	}

	if getUser(ctx) == nil {
		return ctx, sdk.WrapError(sdk.ErrUnauthorized, "Router> Unable to find connected user")
	}

	if rc.Options["needHatchery"] == "true" && getHatchery(ctx) != nil {
		return ctx, nil
	}

	if rc.Options["needWorker"] == "true" {
		permissionOk := api.checkWorkerPermission(ctx, api.mustDB(), rc, mux.Vars(req))
		if !permissionOk {
			return ctx, sdk.WrapError(sdk.ErrForbidden, "Router> Worker not authorized")
		}
		return ctx, nil
	}

	if rc.Options["allowServices"] == "true" && getService(ctx) != nil {
		return ctx, nil
	}

	if getUser(ctx).Admin {
		return ctx, nil
	}

	if rc.Options["needAdmin"] != "true" {
		permissionOk := api.checkPermission(ctx, mux.Vars(req), getPermissionByMethod(req.Method, rc.Options["isExecution"] == "true"))
		if !permissionOk {
			return ctx, sdk.WrapError(sdk.ErrForbidden, "Router> User not authorized")
		}
	} else {
		return ctx, sdk.WrapError(sdk.ErrForbidden, "Router> User not authorized (needAdmin)")
	}

	if rc.Options["needUsernameOrAdmin"] == "true" && getUser(ctx).Username != mux.Vars(req)["username"] {
		// get / update / delete user -> for admin or current user
		// if not admin and currentUser != username in request -> ko
		return ctx, sdk.WrapError(sdk.ErrForbidden, "Router> User not authorized on this resource")
	}

	return ctx, nil
}

func (api *API) deletePermissionMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error) {
	if req.Method == "POST" || req.Method == "PUT" || req.Method == "DELETE" {
		api.deleteUserPermissionCache(ctx, api.Cache)
	}
	return ctx, nil
}

func (api *API) tracingMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error) {
	name := runtime.FuncForPC(reflect.ValueOf(rc.Handler).Pointer()).Name()
	name = strings.Replace(name, ".func1", "", 1)
	name = strings.Replace(name, "github.com/ovh/cds/engine/api.(*API).", "", 1)

	opts := tracing.Options{
		Name:     name,
		Enable:   rc.Options["trace_enable"] == "true",
		Init:     rc.Options["trace_new_trace"] == "true",
		User:     getUser(ctx),
		Worker:   getWorker(ctx),
		Hatchery: getHatchery(ctx),
	}

	ctx, err := tracing.Start(ctx, w, req, opts, api.mustDB(), api.Cache)
	newReq := req.WithContext(ctx)
	*req = *newReq

	return ctx, err
}

func (api *API) tracingPostMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error) {
	ctx, err := tracing.End(ctx, w, req)

	return ctx, err
}
