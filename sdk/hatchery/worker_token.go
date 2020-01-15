package hatchery

import (
	"crypto/rsa"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/ovh/cds/sdk"
)

// NewWorkerToken .
func NewWorkerToken(hatcheryName string, privateKey *rsa.PrivateKey, expiration time.Time, w SpawnArguments) (string, error) {
	claims := WorkerJWTClaims{
		Worker: w,
		StandardClaims: jwt.StandardClaims{
			Issuer:    hatcheryName,
			Subject:   w.WorkerName,
			Id:        sdk.UUID(),
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: expiration.Unix(),
		},
	}

	// FIXME create dedicated struct with only required fields for the token
	claims.Worker.Model = nil

	jwtoken := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	signedJWToken, err := jwtoken.SignedString(privateKey)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	return signedJWToken, nil
}
