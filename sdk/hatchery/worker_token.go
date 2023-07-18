package hatchery

import (
	"crypto/rsa"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt"
	"github.com/ovh/cds/sdk"
)

// NewWorkerToken .
func NewWorkerToken(hatcheryName string, privateKey *rsa.PrivateKey, expiration time.Time, w SpawnArguments) (string, error) {
	jobIDInt, err := strconv.ParseInt(w.JobID, 10, 64)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	claims := WorkerJWTClaims{
		Worker: SpawnArgumentsJWT{
			WorkerName:   w.WorkerName,
			JobID:        jobIDInt,
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
	if w.Model.ModelV1 != nil {
		claims.Worker.Model.ID = w.Model.ModelV1.ID
		claims.Worker.Model.Name = w.Model.ModelV1.Name
	}

	jwtoken := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	signedJWToken, err := jwtoken.SignedString(privateKey)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	return signedJWToken, nil
}

// NewWorkerTokenV2 .
func NewWorkerTokenV2(hatcheryName string, privateKey *rsa.PrivateKey, expiration time.Time, w SpawnArguments) (string, error) {
	claims := WorkerJWTClaimsV2{
		Worker: SpawnArgumentsJWTV2{
			WorkerName:   w.WorkerName,
			RunJobID:        w.JobID,
			HatcheryName: w.HatcheryName,
      ModelName: w.ModelName(),
		},
		StandardClaims: jwt.StandardClaims{
			Issuer:    hatcheryName,
			Subject:   w.WorkerName,
			Id:        sdk.UUID(),
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: expiration.Unix(),
		},
	}
	jwtoken := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	signedJWToken, err := jwtoken.SignedString(privateKey)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	return signedJWToken, nil
}
