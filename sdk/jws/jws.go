package jws

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"

	"gopkg.in/square/go-jose.v2"
)

// NewRandomRSAKey generates a public/private key pair
func NewRandomRSAKey() (*rsa.PrivateKey, error) {
	// Generate a public/private key pair to use for this example.
	return rsa.GenerateKey(rand.Reader, 2048)
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

// Sign a json marshalled content and returns a protected JWS object using the full serialization format.
func Sign(signer jose.Signer, content interface{}) (string, error) {
	btes, err := json.Marshal(content)
	if err != nil {
		return "", err
	}
	object, err := signer.Sign(btes)
	if err != nil {
		return "", err
	}
	return object.CompactSerialize()
}

// Verify parses the serialized, protected JWS object, than verifying the signature on the payload
// and unmarshal the payload into i
func Verify(publicKey *rsa.PublicKey, s string, i interface{}) error {
	object, err := jose.ParseSigned(s)
	if err != nil {
		return err
	}
	output, err := object.Verify(publicKey)
	if err != nil {
		return err
	}
	return json.Unmarshal(output, i)
}

func UnsafeParse(s string, i interface{}) error {
	object, err := jose.ParseSigned(s)
	if err != nil {
		return err
	}
	output := object.UnsafePayloadWithoutVerification()
	return json.Unmarshal(output, i)
}
