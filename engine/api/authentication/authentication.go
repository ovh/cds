package authentication

import (
	"context"
	"crypto/rsa"
	"sort"
	"time"

	jwt "github.com/golang-jwt/jwt"

	"github.com/ovh/cds/engine/authentication"
)

var (
	signers []authentication.Signer
)

type KeyConfig struct {
	Timestamp int64  `toml:"timestamp" mapstructure:"timestamp"`
	Key       string `toml:"key" mapstructure:"key"`
}

// Init the package by passing the signing key
func Init(ctx context.Context, issuer string, keys []KeyConfig) error {
	// sort the keys to set the most recent signer at the end
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Timestamp < keys[j].Timestamp
	})

	signers = make([]authentication.Signer, len(keys))

	for i := range keys {
		s, err := authentication.NewSigner(issuer, []byte(keys[i].Key))
		if err != nil {
			return err
		}
		signers[i] = s
	}

	return nil
}

func getSigner() authentication.Signer {
	if len(signers) == 0 {
		panic("signer is not set")
	}
	return signers[len(signers)-1] // return the most recent signer
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
	var lastError error
	// Check with the most recent signer first
	for i := len(signers) - 1; i >= 0; i-- {
		s := signers[i]
		res, err := s.VerifyJWT(token)
		if err == nil && res != nil {
			return res, nil
		}
		lastError = err
	}
	return nil, lastError
}

func SignJWS(content interface{}, now time.Time, duration time.Duration) (string, error) {
	return getSigner().SignJWS(content, now, duration)
}

func VerifyJWS(signature string, content interface{}) error {
	var lastError error
	// Check with the most recent signer first
	for i := len(signers) - 1; i >= 0; i-- {
		s := signers[i]
		err := s.VerifyJWS(signature, content)
		if err == nil {
			return nil
		}
		lastError = err
	}
	return lastError
}
