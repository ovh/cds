package accesstoken

import (
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/sdk"
)

// NewJWT generate a signed token for given auth session.
func NewJWT(s *sdk.AuthSession) (string, error) {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, sdk.AuthSessionJWTClaims{
		ID:       s.ID,
		GroupIDs: s.GroupIDs,
		Scopes:   s.Scopes,
		StandardClaims: jwt.StandardClaims{
			Issuer:    IssuerName,
			Subject:   s.ConsumerID,
			Id:        s.ID,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: s.ExpireAt.Unix(),
		},
	})
	return SignJWT(jwtToken)
}

// SignJWT returns a jwt signed string using CDS signing key.
func SignJWT(jwtToken *jwt.Token) (string, error) {
	ss, err := jwtToken.SignedString(signingKey)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	return ss, nil
}
