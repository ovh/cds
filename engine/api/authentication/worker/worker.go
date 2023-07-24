package worker

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	jwt "github.com/golang-jwt/jwt"
	"github.com/rockbears/log"

	hatchdao "github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/jws"
)

// SessionDuration the life time of a worker session.
var SessionDuration = 24 * time.Hour

// VerifyToken checks token technical validity
func VerifyToken(ctx context.Context, db gorp.SqlExecutor, s string) (*hatchery.WorkerJWTClaims, error) {
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
		log.Debug(ctx, "worker.VerifyToken> unsafe token is valid - issuer: %v expiresAt: %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
	} else {
		return nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}

	h, err := services.LoadByNameAndType(ctx, db, claims.Worker.HatcheryName, sdk.TypeHatchery)
	if err != nil {
		log.Error(ctx, "worker.VerifyToken> unable to load hatchery %s: %v", claims.Worker.HatcheryName, err)
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
		log.Debug(ctx, "worker.VerifyToken> invalid token parse: %s", s)
		return nil, sdk.NewErrorWithStack(err, sdk.ErrForbidden)
	}

	claims, ok = token.Claims.(*hatchery.WorkerJWTClaims)
	if ok && token.Valid {
		log.Debug(ctx, "worker.VerifyToken> token is valid %v %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
	} else {
		log.Debug(ctx, "worker.VerifyToken> invalid token: %s", s)
		return nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}

	return claims, nil
}

// VerifyTokenV2 checks token technical validity
func VerifyTokenV2(ctx context.Context, db gorp.SqlExecutor, s string) (*hatchery.WorkerJWTClaimsV2, *sdk.Hatchery, error) {
	// First we try to parse the token without checking the its validity.
	// The goal is to be able to get information about the worker
	// We need to know with is the hatchery involved to be able to checks the token signature
	// against the hatchery public key
	unsafeToken, _, err := new(jwt.Parser).ParseUnverified(s, &hatchery.WorkerJWTClaimsV2{})
	if err != nil {
		return nil, nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}

	claims, ok := unsafeToken.Claims.(*hatchery.WorkerJWTClaimsV2)
	if ok {
		log.Debug(ctx, "worker.VerifyTokenV2> unsafe token is valid - issuer: %v expiresAt: %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
	} else {
		return nil, nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}

	h, err := hatchdao.LoadHatcheryByName(ctx, db, claims.Worker.HatcheryName)
	if err != nil {
		log.Error(ctx, "worker.VerifyTokenV2> unable to load hatchery %s: %v", claims.Worker.HatcheryName, err)
		return nil, nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}

	publicKey, err := jws.NewPublicKeyFromPEM(h.PublicKey)
	if err != nil {
		return nil, nil, sdk.WithStack(err)
	}

	token, err := jwt.ParseWithClaims(s, &hatchery.WorkerJWTClaimsV2{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, sdk.NewErrorFrom(sdk.ErrUnauthorized, "Unexpected signing method: %v", token.Header["alg"])
			}
			return publicKey, nil
		})
	if err != nil {
		log.Debug(ctx, "worker.VerifyTokenV2> invalid token parse: %s", s)
		return nil, nil, sdk.NewErrorWithStack(err, sdk.ErrForbidden)
	}

	claims, ok = token.Claims.(*hatchery.WorkerJWTClaimsV2)
	if ok && token.Valid {
		log.Debug(ctx, "worker.VerifyTokenV2> token is valid %v %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
	} else {
		log.Debug(ctx, "worker.VerifyTokenV2> invalid token: %s", s)
		return nil, nil, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}

	return claims, h, nil
}
