package worker

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
)

// VerifyToken checks token technical validity
func VerifyToken(db gorp.SqlExecutor, s string) (*hatchery.WorkerJWTClaims, error) {
	// First we try to parse the token without checking the its validity.
	// The goal is to be able to get information about the worker
	// We need to know with is the hatchery involved to be able to checks the token signature
	// against the hatchery public key
	unsafeToken, _, err := new(jwt.Parser).ParseUnverified(s, &hatchery.WorkerJWTClaims{})
	if err != nil {
		return nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}

	claims, ok := unsafeToken.Claims.(*hatchery.WorkerJWTClaims)
	if ok {
		log.Debug("worker.VerifyToken> unsafe token is valid %v %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
	} else {
		return nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}

	h, err := services.FindByNameAndType(db, claims.Worker.HatcheryName, services.TypeHatchery)
	if err != nil {
		return nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}

	publicKey, err := jws.NewPublicKeyFromPEM(h.PublicKey)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	token, err := jwt.ParseWithClaims(s, &hatchery.WorkerJWTClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, sdk.NewErrorFrom(sdk.ErrUnauthorized, "Unexpected signing method: %v", token.Header["alg"])
			}
			return publicKey, nil
		})
	if err != nil {
		log.Debug("worker.VerifyToken> invalid token parse: %s", s)
		return nil, sdk.NewErrorWithStack(err, sdk.ErrForbidden)
	}

	claims, ok = token.Claims.(*hatchery.WorkerJWTClaims)
	if ok && token.Valid {
		log.Debug("worker.VerifyToken> token is valid %v %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
	} else {
		log.Debug("worker.VerifyToken> invalid token: %s", s)
		return nil, sdk.ErrUnauthorized
	}

	return claims, nil
}
