package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	jwtCookieName  = "jwt_token"
	xsrfHeaderName = "X-XSRF-TOKEN"
	xsrfCookieName = "xsrf_token"
)

func (api *API) authMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	ctx, end := observability.Span(ctx, "router.authMiddleware")
	defer end()

	// Tokens (like izanamy)
	ctx, ok, err := api.authStatusTokenMiddleware(ctx, w, req, rc)
	if err != nil {
		return ctx, sdk.WithStack(err)
	}
	if ok {
		log.Info(ctx, "authMiddleware> authentification granted by token")
		return ctx, nil
	}

	// Check for a JWT in current request and add it to the context
	// If a JWT is given, we also checks that there are a valid session and consumer for it
	ctxWithJWT, err := api.jwtMiddleware(ctx, w, req, rc)
	if err != nil {
		return ctx, err
	}

	var (
		session  *sdk.AuthSession
		consumer *sdk.AuthConsumer
	)

	jwt, ok := ctxWithJWT.Value(contextJWT).(*jwt.Token)
	if ok {
		claims := jwt.Claims.(*sdk.AuthSessionJWTClaims)
		sessionID := claims.StandardClaims.Id
		// Check for session based on jwt from context
		session, err = authentication.CheckSession(ctx, api.mustDB(), sessionID)
		if err != nil {
			log.Warning(ctx, "authMiddleware> cannot find a valid session for given JWT: %v", err)
		}
	}

	if session != nil {
		ctx = context.WithValue(ctxWithJWT, contextSession, session)
		// Load auth consumer for current session in database with authentified user and contacts
		c, err := authentication.LoadConsumerByID(ctx, api.mustDB(), session.ConsumerID,
			authentication.LoadConsumerOptions.WithAuthentifiedUser)
		if err != nil {
			return ctx, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}
		// If the consumer is disabled, return an error
		if c.Disabled {
			return ctx, sdk.WrapError(sdk.ErrUnauthorized, "consumer (%s) is disabled", c.ID)
		}
		// If the driver was disabled for the consumer that was found, ignore it
		if _, ok := api.AuthenticationDrivers[c.Type]; ok {
			// Add contacts for consumer's user
			if err := user.LoadOptions.WithContacts(ctx, api.mustDB(), c.AuthentifiedUser); err != nil {
				return ctx, err
			}

			// Add service for consumer if exists
			s, err := services.LoadByConsumerID(ctx, api.mustDB(), c.ID)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return ctx, err
			}
			c.Service = s

			// Add worker for consumer if exists
			w, err := worker.LoadByConsumerID(ctx, api.mustDB(), c.ID)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return ctx, err
			}
			c.Worker = w

			consumer = c
		}
	}

	if consumer != nil {
		ctx = context.WithValue(ctx, contextAPIConsumer, consumer)

		// Checks scopes, one of expected scopes should be in actual scopes
		// Actual scope empty list means wildcard scope, we don't need to check scopes
		expectedScopes, actualScopes := rc.AllowedScopes, consumer.ScopeDetails
		if len(expectedScopes) > 0 && len(actualScopes) > 0 {
			var found bool
		findScope:
			for i := range expectedScopes {
				for j := range actualScopes {
					if actualScopes[j].Scope == expectedScopes[i] {
						// Check if there are scope details, if yes we should check if current route/method is allowed in restrictions
						if len(actualScopes[j].Endpoints) == 0 {
							found = true
							break findScope
						}

						// if the route is not in current consumer allowed endpoints we should not validate the scope
						if exists, endpoint := actualScopes[j].Endpoints.FindEndpoint(rc.CleanURL); exists &&
							len(endpoint.Methods) == 0 || endpoint.Methods.Contains(rc.Method) {
							found = true
							break findScope
						}
					}
				}
			}
			if !found {
				return ctx, sdk.WrapError(sdk.ErrUnauthorized, "token scopes doesn't match expected: %v", expectedScopes)
			}
		}

		// Check that permission are valid for current route and consumer
		if err := api.checkPermission(ctx, mux.Vars(req), rc.PermissionLevel); err != nil {
			return ctx, err
		}

		jwtFromCookieVal := ctx.Value(contextJWTFromCookie)
		jwtFromCookie, _ := jwtFromCookieVal.(bool)
		if jwtFromCookie {
			ctx, err = api.xsrfMiddleware(ctx, w, req, rc)
			if err != nil {
				return ctx, err
			}
		}
	}

	// If we set Auth(false) on a handler, with should have a consumer in the context if a valid JWT is given
	if rc.NeedAuth && getAPIConsumer(ctx) == nil {
		return ctx, sdk.WithStack(sdk.ErrUnauthorized)
	}

	if rc.NeedAdmin && !isAdmin(ctx) {
		return ctx, sdk.WithStack(sdk.ErrForbidden)
	}

	return ctx, nil
}

// Checks static tokens
func (api *API) authStatusTokenMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, bool, error) {
	if len(rc.AllowedTokens) == 0 {
		return ctx, false, nil
	}
	for _, h := range rc.AllowedTokens {
		log.Debug("authStatusTokenMiddleware> checking allowed token: %v", h)
		headerSplitted := strings.Split(h, ":")
		receivedValue := req.Header.Get(headerSplitted[0])
		if receivedValue != headerSplitted[1] {
			return ctx, false, sdk.WrapError(sdk.ErrUnauthorized, "Router> Authorization denied token on %s %s for %s", req.Method, req.URL, req.RemoteAddr)
		}
	}
	return ctx, true, nil
}

func (api *API) jwtMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	ctx, end := observability.Span(ctx, "router.jwtMiddleware")
	defer end()

	var jwtRaw string
	var jwtFromCookie bool
	// Try to get the jwt from the cookie firstly then from the authorization bearer header, a XSRF token with cookie
	jwtCookie, _ := req.Cookie(jwtCookieName)
	if jwtCookie != nil {
		jwtRaw = jwtCookie.Value
		jwtFromCookie = true
	} else if strings.HasPrefix(req.Header.Get("Authorization"), "Bearer ") {
		jwtRaw = strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	}
	// If no jwt is given, simply return empty context without error
	if jwtRaw == "" {
		return ctx, nil
	}

	jwt, err := authentication.CheckSessionJWT(jwtRaw)
	if err != nil {
		if rc.NeedAuth {
			// If the given JWT is not valid log the error and return
			log.Warning(ctx, "jwtMiddleware> invalid given jwt token [%s]: %+v", req.URL.String(), err)
		}
		return ctx, nil
	}

	ctx = context.WithValue(ctx, contextJWTRaw, jwt)
	ctx = context.WithValue(ctx, contextJWT, jwt)
	ctx = context.WithValue(ctx, contextJWTFromCookie, jwtFromCookie)

	return ctx, nil
}

func (api *API) xsrfMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
	ctx, end := observability.Span(ctx, "router.xsrfMiddleware")
	defer end()

	jwtValue := ctx.Value(contextJWT)
	if jwtValue == nil {
		return ctx, sdk.WithStack(sdk.ErrUnauthorized)
	}

	jwt, ok := jwtValue.(*jwt.Token)
	if !ok {
		return ctx, sdk.WithStack(sdk.ErrUnauthorized)
	}

	claims := jwt.Claims.(*sdk.AuthSessionJWTClaims)
	sessionID := claims.StandardClaims.Id

	xsrfToken := req.Header.Get(xsrfHeaderName)
	existingXSRFToken, existXSRFTokenInCache := authentication.GetSessionXSRFToken(api.Cache, sessionID)

	// If it's not a read request we want to check the xsrf token then generate a new one
	// else if its a read request we want to reuse a cached XSRF token or generate one
	if rc.PermissionLevel > sdk.PermissionRead {
		if !existXSRFTokenInCache || xsrfToken != existingXSRFToken {
			return ctx, sdk.WithStack(sdk.ErrUnauthorized)
		}

		newXSRFToken, err := authentication.NewSessionXSRFToken(api.Cache, sessionID)
		if err != nil {
			return ctx, err
		}
		// Set a cookie with the jwt token
		api.SetCookie(w, xsrfCookieName, newXSRFToken,
			time.Now().Add(time.Duration(authentication.XSRFTokenDuration)*time.Second))
	} else {
		if !existXSRFTokenInCache {
			var err error
			existingXSRFToken, err = authentication.NewSessionXSRFToken(api.Cache, sessionID)
			if err != nil {
				return ctx, err
			}
		}

		// Set a cookie with the jwt token
		api.SetCookie(w, xsrfCookieName, existingXSRFToken,
			time.Now().Add(time.Duration(authentication.XSRFTokenDuration)*time.Second))
	}

	return ctx, nil
}
