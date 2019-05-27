package hatchery

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/ovh/cds/sdk"
)

func NewWorkerToken(h Interface, expiration time.Time, w SpawnArguments) (sdk.AccessToken, string, error) {
	var token sdk.AccessToken
	token.ID = sdk.UUID()
	token.Created = time.Now()
	token.ExpireAt = expiration
	token.Name = w.WorkerName
	token.Origin = h.ServiceName()
	token.Status = sdk.AccessTokenStatusEnabled

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
	signedJWToken, err := jwtoken.SignedString(h.PrivateKey())
	if err != nil {
		return token, "", sdk.WithStack(err)
	}
	return token, signedJWToken, nil
}
