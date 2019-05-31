package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/accesstoken"
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
	ctx, err = api.authJWTMiddleware(ctx, w, req, rc)
	if err != nil {
		return ctx, sdk.WithStack(err)
	}

	token := JWT(ctx)
	if token == nil {
		return ctx, nil
	}

	// Put the granted user in the context
	var APIConsumer = sdk.APIConsumer{
		Fullname:   token.Name,
		OnBehalfOf: token.AuthentifiedUser,
		Groups:     token.Groups,
	}
	ctx = context.WithValue(ctx, contextAPIConsumer, &APIConsumer)

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

	// Checks permissions
	if !rc.NeedAuth {
		return ctx, nil
	}
	if rc.NeedAdmin {
		if !isAdmin(ctx) || !sdk.IsInArray(sdk.AccessTokenScopeAdmin, actualScopes) {
			return ctx, sdk.WithStack(sdk.ErrForbidden)
		}
	}

	if err := api.checkPermission(ctx, mux.Vars(req), rc.PermissionLevel); err != nil {
		return ctx, err
	}

	return ctx, nil
}

// Check Provider
func (api *API) authAllowProviderMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, bool, error) {
	if rc.AllowProvider {
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
			ctx = context.WithValue(ctx, contextProvider, providerName)
			return ctx, false, nil
		}
	}
	return ctx, true, nil
}

// Checks static tokens
func (api *API) authStatusTokenMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, bool, error) {
	for _, h := range rc.AllowedTokens {
		headerSplitted := strings.Split(h, ":")
		receivedValue := req.Header.Get(headerSplitted[0])
		if receivedValue != headerSplitted[1] {
			return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied token on %s %s for %s", req.Method, req.URL, req.RemoteAddr)
		}
	}
	return ctx, true, nil
}

func (api *API) authJWTMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	ctx, end := observability.Span(ctx, "router.authJWTMiddleware")
	defer end()

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

	if jwt == "" {
		if rc.NeedAuth {
			return ctx, sdk.WithStack(sdk.ErrUnauthorized)
		}
		return ctx, nil
	}

	log.Debug("authJWTMiddleware> checking jwt token %s...", jwt[:12])
	ctx = context.WithValue(ctx, contextJWTRaw, jwt)

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

	return context.WithValue(ctx, contextJWT, &token), nil
}
