package worker

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/services"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// VerifyToken checks token technical validity
func VerifyToken(db gorp.SqlExecutor, s string) (*jwt.Token, error) {
	// First we try to parse the token without checking the its validity.
	// The goal is to be able to get information about the worker
	// We need to know with is the hatchery involved to be able to checks the token signature
	// against the hatchery public key
	unsafeToken, _, err := new(jwt.Parser).ParseUnverified(s, &sdk.WorkerJWTClaims{})
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	claims, ok := unsafeToken.Claims.(*sdk.WorkerJWTClaims)
	if ok && unsafeToken.Valid {
		log.Debug("Token isValid %v %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
	} else {
		return nil, sdk.WithStack(sdk.ErrUnauthorized)
	}

	hatchery, err := services.FindByID(db, claims.Worker.HatcheryID)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(hatchery.PublicKey)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	token, err := jwt.ParseWithClaims(s, &sdk.WorkerJWTClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, sdk.NewErrorFrom(sdk.ErrUnauthorized, "Unexpected signing method: %v", token.Header["alg"])
			}
			return publicKey, nil
		})
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	if claims, ok := token.Claims.(*sdk.WorkerJWTClaims); ok && token.Valid {
		log.Debug("Token isValid %v %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
	} else {
		return nil, sdk.ErrUnauthorized
	}

	return token, nil
}
