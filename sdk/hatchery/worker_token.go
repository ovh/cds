package hatchery

import (
	"crypto/rsa"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/ovh/cds/sdk"
)

func NewWorkerToken(hatcheryName string, privateKey *rsa.PrivateKey, expiration time.Time, w SpawnArguments) (sdk.AuthSession, string, error) {
	var token sdk.AuthSession
	token.ID = sdk.UUID()
	token.Created = time.Now()
	token.ExpireAt = expiration
	token.Scopes = []string{sdk.AccessTokenScopeWorker}
	claims := WorkerJWTClaims{
		Worker: w,
		StandardClaims: jwt.StandardClaims{
			Issuer:    hatcheryName,
			Subject:   w.WorkerName,
			Id:        token.ID,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: token.ExpireAt.Unix(),
		},
	}

	jwtoken := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	signedJWToken, err := jwtoken.SignedString(privateKey)
	if err != nil {
		return token, "", sdk.WithStack(err)
	}
	return token, signedJWToken, nil
}
