package accesstoken

import (
	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// VerifyJWT .
func VerifyJWT(jwtToken string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(jwtToken, &sdk.AuthSessionJWTClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, sdk.NewErrorFrom(sdk.ErrUnauthorized, "unexpected signing method: %v", token.Header["alg"])
			}
			return verifyKey, nil
		})
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	if claims, ok := token.Claims.(*sdk.AuthSessionJWTClaims); ok && token.Valid {
		log.Debug("authentication.jwtVerify> jwt token is valid: %v %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
		return token, nil
	}

	return nil, sdk.WithStack(sdk.ErrUnauthorized)
}

// VerifySession .
func VerifySession(ctx context.Context, db gorp.SqlExecutor, sessionID string) (*sdk.AuthSession, error) {
	// Load the session from the id read in the claim
	session, err := LoadSessionByID(ctx, db, sessionID)
	if err != nil {
		return nil, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrUnauthorized, "cannot load session for id: %s", sessionID))
	}
	if session == nil {
		log.Debug("authentication.sessionMiddleware> no session found for id: %s", sessionID)
		return nil, sdk.WithStack(sdk.ErrUnauthorized)
	}

	// TODO check session validity

	return session, nil
}
