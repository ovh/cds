package authentication

import (
	"crypto/rsa"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
)

var (
	IssuerName string
	signingKey *rsa.PrivateKey
	verifyKey  *rsa.PublicKey
)

// Init the package by passing the signing key
func Init(issuer string, k []byte) error {
	IssuerName = issuer

	var err error
	signingKey, err = jwt.ParseRSAPrivateKeyFromPEM(k)
	if err != nil {
		return sdk.WithStack(err)
	}
	verifyKey = &signingKey.PublicKey

	return nil
}

func GetSigningKey() *rsa.PrivateKey {
	if signingKey == nil {
		panic("signing rsa private key is not set")
	}
	return signingKey
}

// SignJWT returns a jwt string using CDS signing key.
func SignJWT(jwtToken *jwt.Token) (string, error) {
	ss, err := jwtToken.SignedString(signingKey)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	return ss, nil
}

// VerifyJWT func is used when parsing a jwt token to validate signature.
func VerifyJWT(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, sdk.NewErrorFrom(sdk.ErrUnauthorized, "unexpected signing method: %v", token.Header["alg"])
	}
	return verifyKey, nil
}

// SignJWS returns a jws string using CDS signing key.
func SignJWS(content interface{}) (string, error) {
	signer, err := jws.NewSigner(signingKey)
	if err != nil {
		return "", err
	}
	signature, err := jws.Sign(signer, content)
	if err != nil {
		return "", err
	}
	return signature, nil
}

// VerifyJWS checks the validity of given jws string with CDS signing key.
func VerifyJWS(signature string, content interface{}) error {
	return jws.Verify(verifyKey, signature, content)
}
