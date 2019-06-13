package api

import (
	"context"
	"net/http"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	jwtCookieName  = "jwt_token"
	xsrfHeaderName = "X-XSRF-TOKEN"
)

func (api *API) authMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	// If the route don't need auth return directly
	if !rc.NeedAuth {
		return ctx, nil
	}

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

	// Check for a JWT in current request and add it to the context
	ctx, err = api.jwtMiddleware(ctx, req, rc)
	if err != nil {
		return ctx, err
	}

	jwt, ok := ctx.Value(contextJWT).(*jwt.Token)
	if !ok {
		return nil, sdk.WithStack(sdk.ErrUnauthorized)
	}
	claims := jwt.Claims.(sdk.AuthSessionJWTClaims)
	sessionID := claims.StandardClaims.Id

	// Check for session based on jwt from context
	session, err := authentication.CheckSession(ctx, api.mustDB(), sessionID)
	if err != nil {
		return ctx, err
	}

	ctx = context.WithValue(ctx, contextSession, session)

	// Load auth consumer for current session in database
	consumer, err := authentication.LoadConsumerByID(ctx, api.mustDB(), session.ConsumerID,
		authentication.LoadConsumerOptions.WithAuthentifiedUser)
	if err != nil {
		return ctx, err
	}
	if consumer == nil {
		return ctx, sdk.WithStack(sdk.ErrUnauthorized)
	}

	ctx = context.WithValue(ctx, contextAPIConsumer, consumer)

	// Checks scopes
	expectedScopes := getHandlerScope(ctx)
	actualScopes := consumer.Scopes

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

func (api *API) jwtMiddleware(ctx context.Context, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	ctx, end := observability.Span(ctx, "router.authJWTMiddleware")
	defer end()

	var jwtRaw string
	var xsrfTokenNeeded bool

	log.Debug("authJWTMiddleware> searching for a jwt token")

	// Try to get the jwt from the cookie firstly then from the authorization bearer header, a XSRF token with cookie
	jwtCookie, _ := req.Cookie(jwtCookieName)
	if jwtCookie != nil {
		jwtRaw = jwtCookie.Value
		xsrfTokenNeeded = true
	} else if strings.HasPrefix(req.Header.Get("Authorization"), "Bearer ") {
		jwtRaw = strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	}
	if jwtRaw == "" {
		return ctx, sdk.WithStack(sdk.ErrUnauthorized)
	}

	log.Debug("authJWTMiddleware> checking jwt token %s...", jwtRaw[:12])

	jwt, err := authentication.CheckSessionJWT(jwtRaw)
	if err != nil {
		return ctx, err
	}
	claims := jwt.Claims.(sdk.AuthSessionJWTClaims)
	sessionID := claims.StandardClaims.Id

	// Checking X-XSRF-TOKEN header
	if xsrfTokenNeeded {
		log.Debug("authJWTMiddleware> searching for a xsrf token")

		xsrfToken := req.Header.Get(xsrfHeaderName)

		log.Debug("authJWTMiddleware> checking xsrf token %s...", xsrfToken[:12])

		if !authentication.CheckXSRFToken(api.Cache, sessionID, xsrfToken) {
			return ctx, sdk.WithStack(sdk.ErrUnauthorized)
		}
	}

	ctx = context.WithValue(ctx, contextJWTRaw, jwt)
	ctx = context.WithValue(ctx, contextJWT, jwt)

	return ctx, nil
}
