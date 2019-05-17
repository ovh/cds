package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Check Provider
func (api *API) authAllowProviderMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, bool, error) {
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
			return ctx, false, nil
		}
	}
	return ctx, true, nil
}

// Checks static tokens
func (api *API) authStatusTokenMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, bool, error) {
	if h, ok := rc.Options["token"]; ok {
		headerSplitted := strings.Split(h, ":")
		receivedValue := req.Header.Get(headerSplitted[0])
		if receivedValue != headerSplitted[1] {
			return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied token on %s %s for %s", req.Method, req.URL, req.RemoteAddr)
		}
		return ctx, false, nil
	}
	return ctx, true, nil
}

// Old and Deprecated Authentication
func (api *API) authDeprecatedMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, bool, error) {
	headers := req.Header

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

	//Get the permission for either the hatchery, the worker or the user
	switch {
	case getProvider(ctx) != nil:
	case getHatchery(ctx) != nil:
		h := getHatchery(ctx)
		if h != nil && h.GroupID != nil {
			g, perm, err := loadPermissionsByGroupID(api.mustDB(), api.Cache, *h.GroupID)
			if err != nil {
				return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> cannot load group permissions for GroupID %d err:%s", *h.GroupID, err)
			}
			deprecatedGetUser(ctx).Permissions = perm
			deprecatedGetUser(ctx).Groups = append(deprecatedGetUser(ctx).Groups, g)
		}
	case getWorker(ctx) != nil:
		//Refresh the worker
		workerCtx := getWorker(ctx)
		if err := worker.RefreshWorker(api.mustDB(), workerCtx); err != nil {
			return ctx, false, sdk.WrapError(err, "Unable to refresh worker")
		}

		if workerCtx.ModelID != 0 {
			// worker have a model, load model, then load model's group
			m, err := worker.LoadWorkerModelByID(api.mustDB(), workerCtx.ModelID)
			if err != nil {
				return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> cannot load worker: %s - name:%s modelID:%d", err, workerCtx.Name, workerCtx.ModelID)
			}

			if m.GroupID == group.SharedInfraGroup.ID {
				// it's a shared.infra model, load group from token only: workerCtx.GroupID
				if err := api.deprecatedSetGroupsAndPermissionsFromGroupID(ctx, workerCtx.GroupID); err != nil {
					return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> model shared.infra:%d", m.GroupID)
				}
			} else {
				// this model is not attached to shared.infra group, load group with m.GroupID
				if err := api.deprecatedSetGroupsAndPermissionsFromGroupID(ctx, m.GroupID); err != nil {
					return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> model not shared.infra:%d", m.GroupID)
				}
			}
		} else {
			// worker does not have a model, take group from token only
			if err := api.deprecatedSetGroupsAndPermissionsFromGroupID(ctx, workerCtx.GroupID); err != nil {
				return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> no model, worker not shared.infra:%d", workerCtx.GroupID)
			}
		}
	case deprecatedGetUser(ctx) != nil:
		// TEMPORARY CODE, IT SHOULD BE REMOVED WHEN ALL WILL BE MIGRATED TO JWT TOKENS
		u := deprecatedGetUser(ctx)
		authUser, err := user.LoadByOldUserID(api.mustDB(), u.ID)
		if err != nil {
			return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Unable to load authentified user from ID %d: %v", deprecatedGetUser(ctx).ID, err)
		}
		if err := loadUserPermissions(api.mustDB(), api.Cache, authUser); err != nil {
			return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Unable to load user %d permission: %v", deprecatedGetUser(ctx).ID, err)
		}
		var grantedUser = sdk.GrantedUser{
			Fullname:   u.Fullname,
			OnBehalfOf: *authUser,
			Groups:     u.Groups,
		}
		ctx = context.WithValue(ctx, auth.ContextGrantedUser, &grantedUser)
		// TEMPORARY CODE - END
	}

	if rc.Options["auth"] != "true" {
		return ctx, false, nil
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

func (api *API) authJWTMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, bool, error) {
	token := JWT(ctx)
	if token == nil {
		return ctx, true, nil
	}

	// Put the granted user in the context
	var grantedUser = sdk.GrantedUser{
		Fullname:   token.Description,
		OnBehalfOf: token.AuthentifiedUser,
		Groups:     token.Groups,
	}
	ctx = context.WithValue(ctx, auth.ContextGrantedUser, &grantedUser)

	// TEMPORARY CODE
	// SHOULD BE REMOVED WITH REFACTO OF PERMISSIONS
	var err error
	ctx, err = auth.Session_DEPRECATED(ctx, api.mustDB(), token.ID, token.AuthentifiedUser.Username)
	if err != nil {
		return ctx, false, sdk.WithStack(err)
	}

	for _, g := range grantedUser.Groups {
		if err := api.deprecatedSetGroupsAndPermissionsFromGroupID(ctx, g.ID); err != nil {
			return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Unable to load user %d permission: %v", deprecatedGetUser(ctx).ID, err)
		}
	}
	// END OF TEMPORARY CODE

	return ctx, false, nil
}
