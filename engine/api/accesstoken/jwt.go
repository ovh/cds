package accesstoken

import (
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/sdk"
)

// NewJWT generate a signed token for given auth session.
func NewJWT(s *sdk.AuthSession) (string, error) {
	claims := sdk.AuthSessionJWTClaims{
		ID:       s.ID,
		GroupIDs: s.GroupIDs,
		Scopes:   s.Scopes,
		StandardClaims: jwt.StandardClaims{
			Issuer:    LocalIssuer,
			Subject:   s.ConsumerID,
			Id:        s.ID,
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: s.ExpireAt.Unix(),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)

	ss, err := jwtToken.SignedString(signingKey)
	if err != nil {
		return "", sdk.WithStack(err)
	}

	return ss, nil
}
