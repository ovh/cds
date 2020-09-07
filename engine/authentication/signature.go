package authentication

import (
	"crypto/rsa"
	"encoding/json"
	"reflect"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
)

var _ Signer = new(signer)
var _ Verifier = new(verifier)

type Signer interface {
	Verifier
	GetIssuerName() string
	GetSigningKey() *rsa.PrivateKey
	SignJWT(jwtToken *jwt.Token) (string, error)
	SignJWS(content interface{}, duration time.Duration) (string, error)
}

type Verifier interface {
	VerifyJWT(token *jwt.Token) (interface{}, error)
	VerifyJWS(signature string, content interface{}) error
	GetVerifyKey() *rsa.PublicKey
}

func NewSigner(issuer string, k []byte) (Signer, error) {
	s := signer{
		issuerName: issuer,
	}

	var err error
	s.signingKey, err = jwt.ParseRSAPrivateKeyFromPEM(k)
	if err != nil {
		return s, sdk.WithStack(err)
	}
	s.verifyKey = &s.signingKey.PublicKey

	return s, nil
}

func NewVerifier(k *rsa.PublicKey) verifier {
	return verifier{
		verifyKey: k,
	}
}

type signer struct {
	verifier
	issuerName string
	signingKey *rsa.PrivateKey
}

type verifier struct {
	verifyKey *rsa.PublicKey
}

func (s signer) GetIssuerName() string {
	return s.issuerName
}

func (s signer) GetSigningKey() *rsa.PrivateKey {
	return s.signingKey
}

// SignJWT returns a jwt string using CDS signing key.
func (s signer) SignJWT(jwtToken *jwt.Token) (string, error) {
	ss, err := jwtToken.SignedString(s.signingKey)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	return ss, nil
}

// VerifyJWT func is used when parsing a jwt token to validate signature.
func (v verifier) VerifyJWT(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, sdk.NewErrorFrom(sdk.ErrUnauthorized, "unexpected signing method: %v", token.Header["alg"])
	}
	return v.verifyKey, nil
}

func (v verifier) GetVerifyKey() *rsa.PublicKey {
	return v.verifyKey
}

// signaturePayload contains fields for a jws signature payload.
type signaturePayload struct {
	Type   string                 `json:"type"`
	Expire int64                  `json:"expire"`
	Data   map[string]interface{} `json:"data"`
}

// SignJWS returns a jws string using CDS signing key.
func (s signer) SignJWS(content interface{}, duration time.Duration) (string, error) {
	buf, err := json.Marshal(content)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	var jsonData map[string]interface{}
	if err := json.Unmarshal(buf, &jsonData); err != nil {
		return "", sdk.WithStack(err)
	}

	payload := signaturePayload{
		Type: payloadDataType(content),
		Data: jsonData,
	}
	if duration > 0 {
		payload.Expire = time.Now().Add(duration).Unix()
	}

	signer, err := jws.NewSigner(s.signingKey)
	if err != nil {
		return "", err
	}
	signature, err := jws.Sign(signer, payload)
	if err != nil {
		return "", err
	}
	return signature, nil
}

// VerifyJWS checks the validity of given jws string with CDS signing key.
func (s verifier) VerifyJWS(signature string, content interface{}) error {
	var payload signaturePayload
	if err := jws.Verify(s.verifyKey, signature, &payload); err != nil {
		return err
	}

	if payload.Type != payloadDataType(content) || (payload.Expire > 0 && payload.Expire < time.Now().Unix()) {
		return sdk.NewErrorFrom(sdk.ErrUnauthorized, "invalid given jws token or expired: %+v", payload)
	}

	buf, err := json.Marshal(payload.Data)
	if err != nil {
		return sdk.WrapError(err, "unable to decode payload data")
	}
	if err := json.Unmarshal(buf, content); err != nil {
		return sdk.WrapError(err, "unable to decode payload data")
	}

	return nil
}

func payloadDataType(content interface{}) string {
	t := reflect.TypeOf(content)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}
