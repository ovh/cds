package main

import (
	"crypto/rsa"

	jwt "github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
)

var (
	signingKey *rsa.PrivateKey
	verifyKey  *rsa.PublicKey
)

func InitJWT(k []byte) error {
	var err error
	signingKey, err = jwt.ParseRSAPrivateKeyFromPEM(k)
	if err != nil {
		return errors.WithStack(err)
	}
	verifyKey = &signingKey.PublicKey
	return nil
}

func sign(jwtToken *jwt.Token) (string, error) {
	ss, err := jwtToken.SignedString(signingKey)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return ss, nil
}

func verify(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
	}
	return verifyKey, nil
}
