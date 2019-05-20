package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/ovh/cds/engine/api/auth"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) jwtMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	var jwt string
	var xsrfToken string

	// Try to load the token from the cookie or from the authorisation bearer header
	jwtCookie, _ := req.Cookie("jwt_token")
	if jwtCookie != nil {
		log.Debug("ajwtMiddleware> reading jwt token cookie")
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
		log.Debug("ajwtMiddleware> skipping jwt token verification")
		return ctx, nil
	}

	log.Debug("ajwtMiddleware> checking jwt token %s...", jwt[:12])

	ctx, end := observability.Span(ctx, "router.authJWTMiddleware")
	defer end()

	// Get the access token
	token, valid, err := accesstoken.IsValid(api.mustDB(), jwt)
	if err != nil {
		return ctx, err
	}

	// Observability tags
	observability.Current(ctx, observability.Tag(observability.TagToken, token.ID))

	// Is the jwttoken was not valid: raised an error
	if !valid {
		return ctx, sdk.WithStack(sdk.ErrUnauthorized)
	}

	// Checks XSRF token only from token coming from UI
	if token.Origin == accesstoken.OriginUI {
		if !accesstoken.CheckXSRFToken(api.Cache, token, xsrfToken) {
			return ctx, sdk.WithStack(sdk.ErrUnauthorized)
		}
	}

	// Checks scopes
	expectedScopes := getHandlerScope(ctx)
	actualScopes := token.Scopes

	var scopeOK bool
	for _, s := range actualScopes {
		if s == sdk.AccessTokenScopeALL || sdk.IsInArray(s, expectedScopes) {
			scopeOK = true
			break
		}
	}
	if !scopeOK {
		return ctx, sdk.WrapError(sdk.ErrUnauthorized, "token scope (%v) doesn't match (%v)", actualScopes, expectedScopes)
	}

	return context.WithValue(ctx, auth.ContextJWT, &token), nil
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
