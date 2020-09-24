package jws

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"

	"gopkg.in/square/go-jose.v2"

	"github.com/ovh/cds/sdk"
)

func NewRandomSymmetricKey(size int) ([]byte, error) {
	if size <= 0 || size%8 != 0 {
		return nil, sdk.WithStack(fmt.Errorf("invalid key size"))
	}

	k := make([]byte, size)
	if _, err := rand.Read(k); err != nil {
		return nil, sdk.WithStack(err)
	}
	return k, nil
}

// NewRandomRSAKey generates a public/private key pair
func NewRandomRSAKey() (*rsa.PrivateKey, error) {
	// Generate a public/private key pair to use for this example.
	return rsa.GenerateKey(rand.Reader, 2048)
}

func ExportPrivateKey(pk *rsa.PrivateKey) ([]byte, error) {
	var pemPrivateBlock = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(pk),
	}
	buffer := new(bytes.Buffer)
	if err := pem.Encode(buffer, pemPrivateBlock); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func ExportPublicKey(pk *rsa.PrivateKey) ([]byte, error) {
	var pemPublicBlock = &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&pk.PublicKey),
	}
	buffer := new(bytes.Buffer)
	if err := pem.Encode(buffer, pemPublicBlock); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func NewPublicKeyFromPEM(pk []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pk)
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		return nil, errors.New("failed to decode PEM block containing public key")
	}

	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pub, nil
}

// NewSigner instantiate a signer using RSASSA-PSS (SHA512) with the given private key.
func NewSigner(privateKey *rsa.PrivateKey) (jose.Signer, error) {
	// Instantiate a signer using RSASSA-PSS (SHA512) with the given private key.
	return jose.NewSigner(jose.SigningKey{Algorithm: jose.PS512, Key: privateKey}, nil)
}

// NewHMacSigner instantiates a signer using HMAC using SHA-512 with the given private key.
func NewHMacSigner(secret []byte) (jose.Signer, error) {
	sign, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS512, Key: secret}, nil)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	return sign, nil
}

// Sign a json marshalled content and returns a protected JWS object using the full serialization format.
func Sign(signer jose.Signer, content interface{}) (string, error) {
	btes, err := json.Marshal(content)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	object, err := signer.Sign(btes)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	compact, err := object.CompactSerialize()
	if err != nil {
		return "", sdk.WithStack(err)
	}
	return compact, nil
}

// Verify parses the serialized, protected JWS object, than verifying the signature on the payload
// and unmarshal the payload into i
func Verify(key interface{}, s string, i interface{}) error {
	object, err := jose.ParseSigned(s)
	if err != nil {
		if object != nil {
			return sdk.WithStack(fmt.Errorf("unable to parse payload: %v signature: %v", string(object.UnsafePayloadWithoutVerification()), err))
		}
		return sdk.WithStack(err)
	}
	output, err := object.Verify(key)
	if err != nil {
		return sdk.WithStack(fmt.Errorf("unable to verify payload: %v signature: %v", string(object.UnsafePayloadWithoutVerification()), err))
	}
	return sdk.WithStack(json.Unmarshal(output, i))
}

func UnsafeParse(s string, i interface{}) error {
	object, err := jose.ParseSigned(s)
	if err != nil {
		return sdk.WithStack(err)
	}
	output := object.UnsafePayloadWithoutVerification()
	return sdk.WithStack(json.Unmarshal(output, i))
}
