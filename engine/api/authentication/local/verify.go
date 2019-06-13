package local

import (
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func GenerateVerifyToken(consumerID string) (string, error) {
	issuedAt := time.Now()
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.StandardClaims{
		Issuer:    authentication.IssuerName,
		Subject:   consumerID,
		IssuedAt:  issuedAt.Unix(),
		ExpiresAt: issuedAt.Add(24 * time.Hour).Unix(),
	})
	return authentication.SignJWT(jwtToken)
}

func CheckVerifyToken(jwtToken string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(jwtToken, jwt.StandardClaims{}, authentication.VerifyJWT)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	if claims, ok := token.Claims.(jwt.StandardClaims); ok && token.Valid {
		log.Debug("localauthentication.CheckVerifyToken> jwt token is valid: %v %v", claims.Issuer, claims.ExpiresAt)
		return token, nil
	}

	return nil, sdk.WithStack(sdk.ErrUnauthorized)
}
