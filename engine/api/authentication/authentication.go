package authentication

import (
	"crypto/rsa"
	"encoding/json"
	"reflect"
	"time"

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
	ss, err := jwtToken.SignedString(GetSigningKey())
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

// signaturePayload contains fields for a jws signature payload.
type signaturePayload struct {
	Type   string                 `json:"type"`
	Expire int64                  `json:"expire"`
	Data   map[string]interface{} `json:"data"`
}

// SignJWS returns a jws string using CDS signing key.
func SignJWS(content interface{}, duration time.Duration) (string, error) {
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

	signer, err := jws.NewSigner(GetSigningKey())
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
func VerifyJWS(signature string, content interface{}) error {
	var payload signaturePayload
	if err := jws.Verify(verifyKey, signature, &payload); err != nil {
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

// VerifyJWSWithSpecificKey checks the validity of given jws string and the public key.
func VerifyJWSWithSpecificKey(publicKey *rsa.PublicKey, signature string, content interface{}) error {
	var payload signaturePayload
	if err := jws.Verify(publicKey, signature, &payload); err != nil {
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
