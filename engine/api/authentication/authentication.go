package authentication

import (
	"crypto/rsa"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/engine/authentication"
)

var (
	signer *authentication.Signer
)

// Init the package by passing the signing key
func Init(issuer string, k []byte) error {
	s, err := authentication.NewSigner(issuer, k)
	if err != nil {
		return err
	}
	signer = &s
	return nil
}

func getSigner() authentication.Signer {
	if signer == nil {
		panic("signer is not set")
	}
	return *signer
}

func GetIssuerName() string {
	return getSigner().GetIssuerName()
}

func GetSigningKey() *rsa.PrivateKey {
	return getSigner().GetSigningKey()
}

func SignJWT(jwtToken *jwt.Token) (string, error) {
	return getSigner().SignJWT(jwtToken)
}

func VerifyJWT(token *jwt.Token) (interface{}, error) {
	return getSigner().VerifyJWT(token)
}

func SignJWS(content interface{}, duration time.Duration) (string, error) {
	return getSigner().SignJWS(content, duration)
}

func VerifyJWS(signature string, content interface{}) error {
	return getSigner().VerifyJWS(signature, content)
}
