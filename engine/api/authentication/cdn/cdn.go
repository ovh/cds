package worker

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// SessionDuration the life time of a CDN token request.
var SessionDuration = 1 * time.Hour

// VerifyToken checks token technical validity
func VerifyToken(publicKey *rsa.PublicKey, tokenStr string) (*sdk.CDNJWTClaims, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("cannot verify token with a nil public key")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &sdk.CDNJWTClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, sdk.NewErrorFrom(sdk.ErrUnauthorized, "Unexpected signing method: %v", token.Header["alg"])
			}
			return publicKey, nil
		})
	if err != nil {
		log.Debug("cdn.VerifyToken> invalid token parse: %s", tokenStr)
		return nil, sdk.NewErrorWithStack(err, sdk.ErrForbidden)
	}

	claims, ok := token.Claims.(*sdk.CDNJWTClaims)
	if ok && token.Valid {
		log.Debug("cdn.VerifyToken> token is valid %v %v", claims.Issuer, claims.StandardClaims.ExpiresAt)
	} else {
		log.Debug("cdn.VerifyToken> invalid token: %s", tokenStr)
		return nil, sdk.ErrUnauthorized
	}

	return claims, nil
}
