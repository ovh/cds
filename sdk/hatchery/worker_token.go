package hatchery

import (
	"crypto/rsa"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/ovh/cds/sdk"
)

// NewWorkerToken .
func NewWorkerToken(hatcheryName string, privateKey *rsa.PrivateKey, expiration time.Time, w SpawnArguments) (string, error) {
	claims := WorkerJWTClaims{
		Worker: SpawnArgumentsJWT{
			WorkerName:   w.WorkerName,
			JobID:        w.JobID,
			RegisterOnly: w.RegisterOnly,
			HatcheryName: w.HatcheryName,
		},
		StandardClaims: jwt.StandardClaims{
			Issuer:    hatcheryName,
			Subject:   w.WorkerName,
			Id:        sdk.UUID(),
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: expiration.Unix(),
		},
	}
	if w.Model != nil {
		claims.Worker.Model.ID = w.Model.ID
		claims.Worker.Model.Name = w.Model.Name
	}

	jwtoken := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	signedJWToken, err := jwtoken.SignedString(privateKey)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	return signedJWToken, nil
}
