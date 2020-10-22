package service

import (
	"context"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const JWTCookieName = "jwt_token"

type contextKey int

const (
	ContextJWT contextKey = iota
	ContextJWTRaw
	ContextJWTFromCookie
	ContextSessionID
)

func NoAuthMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig) (context.Context, error) {
	return ctx, nil
}

func JWTMiddleware(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *HandlerConfig, keyFunc jwt.Keyfunc) (context.Context, error) {
	var jwtRaw string
	var jwtFromCookie bool
	// Try to get the jwt from the cookie firstly then from the authorization bearer header, a XSRF token with cookie
	jwtCookie, _ := req.Cookie(JWTCookieName)
	if jwtCookie != nil {
		jwtRaw = jwtCookie.Value
		jwtFromCookie = true
	} else if strings.HasPrefix(req.Header.Get("Authorization"), "Bearer ") {
		jwtRaw = strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer ")
	}
	// If no jwt is given, simply return empty context without error
	if jwtRaw == "" {
		log.Debug("service.JWTMiddleware> no jwt token found in request")
		return ctx, nil
	}

	jwt, claims, err := CheckSessionJWT(jwtRaw, keyFunc)
	if err != nil {
		// If the given JWT is not valid log the error and return
		log.Warning(ctx, "service.JWTMiddleware> invalid given jwt token [%s]: %+v", req.URL.String(), err)
		return ctx, nil
	}

	ctx = context.WithValue(ctx, ContextJWTRaw, jwt)
	ctx = context.WithValue(ctx, ContextJWT, jwt)
	ctx = context.WithValue(ctx, ContextJWTFromCookie, jwtFromCookie)
	ctx = context.WithValue(ctx, ContextSessionID, claims.ID)

	return ctx, nil
}

// CheckSessionJWT validate given session jwt token.
func CheckSessionJWT(jwtToken string, keyFunc jwt.Keyfunc) (*jwt.Token, *sdk.AuthSessionJWTClaims, error) {
	token, err := jwt.ParseWithClaims(jwtToken, &sdk.AuthSessionJWTClaims{}, keyFunc)
	if err != nil {
		return nil, nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}

	claims, ok := token.Claims.(*sdk.AuthSessionJWTClaims)
	if ok && token.Valid {
		return token, claims, nil
	}

	return nil, nil, sdk.WithStack(sdk.ErrUnauthorized)
}

func OverrideAuth(m Middleware) HandlerConfigParam {
	return func(rc *HandlerConfig) {
		rc.OverrideAuthMiddleware = m
	}
}
