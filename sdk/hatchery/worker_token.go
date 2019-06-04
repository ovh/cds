package hatchery

import (
	"crypto/rsa"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/ovh/cds/sdk"
)

func NewWorkerToken(hatcheryName string, privateKey *rsa.PrivateKey, maintainer sdk.AuthentifiedUser, expiration time.Time, w SpawnArguments) (sdk.AccessToken, string, error) {
	var token sdk.AccessToken
	token.ID = sdk.UUID()
	token.Created = time.Now()
	token.ExpireAt = expiration
	token.Name = w.WorkerName
	token.Origin = hatcheryName
	token.Status = sdk.AccessTokenStatusEnabled
	token.AuthentifiedUser = &maintainer
	token.AuthentifiedUserID = maintainer.ID
	token.Scopes = []string{sdk.AccessTokenScopeWorker}
	claims := WorkerJWTClaims{
		Worker: w,
		StandardClaims: jwt.StandardClaims{
			Issuer:    token.Origin,
			Subject:   token.Name,
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
