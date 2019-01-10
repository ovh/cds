package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type contextKey int

const (
	ContextGrantedUser contextKey = iota
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
			var err error
			ctx, err = auth.CheckWorkerAuth(ctx, api.mustDB(), api.Cache, headers)
			if err != nil {
				return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s sdk.WorkerAgent agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		case sdk.ServiceAgent:
			var err error
			ctx, err = auth.CheckServiceAuth(ctx, api.mustDB(), api.Cache, headers)
			if err != nil {
				return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied on %s %s for %s sdk.ServiceAgent agent %s : %s", req.Method, req.URL, req.RemoteAddr, getAgent(req), err)
			}
		default:
			var err error
			ctx, err = api.Router.AuthDriver.CheckAuth(ctx, w, req)
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
		if err := loadUserPermissions(api.mustDB(), api.Cache, deprecatedGetUser(ctx)); err != nil {
			return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Unable to load user %d permission: %v", deprecatedGetUser(ctx).ID, err)
		}
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
		return ctx, false, sdk.WrapError(sdk.ErrForbidden, "Router> Need worker")
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
		permissionOk := api.checkPermission(ctx, mux.Vars(req), getPermissionByMethod(req.Method, rc.Options["isExecution"] == "true"))
		if !permissionOk {
			return ctx, false, sdk.WrapError(sdk.ErrForbidden, "Router> User not authorized")
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
	var jwt string
	var xsrfToken string

	// Try to load the token from the cookie or from the authorisation bearer header
	jwtCookie, _ := req.Cookie("jwt_token")
	if jwtCookie != nil {
		jwt = jwtCookie.Value
		// Checking X-XSRF-TOKEN header if the token is used from a cookie
		xsrfToken = req.Header.Get("X-XSRF-TOKEN")
	} else {
		if strings.HasPrefix(req.Header.Get("Authorization"), "Bearer ") {
			jwt = strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
		}
	}

	// For now if there is no JWT token fallback to deprecated code
	if jwt == "" {
		log.Debug("api.authJWTMiddleware> skipping jwt token verification")
		return ctx, true, nil
	}

	log.Debug("api.authJWTMiddleware> checking jwt token %s...", jwt[:12])

	ctx, end := observability.Span(ctx, "router.authJWTMiddleware")
	defer end()

	// Get the access token
	token, valid, err := accesstoken.IsValid(api.mustDB(), jwt)
	if err != nil {
		return ctx, false, err
	}

	// Observability tags
	observability.Current(ctx, observability.Tag(observability.TagToken, token.ID))

	// Is the jwttoken was not valid: raised an error
	if !valid {
		return ctx, false, sdk.WithStack(sdk.ErrUnauthorized)
	}

	// Checks XSRF token only from token coming from UI
	if token.Origin == accesstoken.OriginUI {
		if !accesstoken.CheckXSRFToken(api.Cache, token, xsrfToken) {
			return ctx, false, sdk.WithStack(sdk.ErrUnauthorized)
		}
	}

	// Put the granted user in the context
	var grantedUser = sdk.GrantedUser{
		Fullname:   token.Description,
		OnBehalfOf: token.User,
		Groups:     token.Groups,
	}
	ctx = context.WithValue(ctx, ContextGrantedUser, &grantedUser)

	// TEMPORARY CODE
	// SHOULD BE REMOVED WITH REFACTO OF PERMISSIONS
	ctx, err = api.Router.AuthDriver.DeprecatedSession(ctx, token.ID, token.User.Username)
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
