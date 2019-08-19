package corpsso

import (
	"encoding/json"
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gpg"

	jose "gopkg.in/square/go-jose.v2"
)

type nonceSource struct{}

func (n nonceSource) Nonce() (string, error) {
	return sdk.UUID(), nil
}

var nonceSrc jose.NonceSource = nonceSource{}

func prepareRequest(requestSigningKey string, request interface{}) (string, error) {
	pk, err := gpg.NewPrivateKeyFromPem(requestSigningKey, "")
	if err != nil {
		return "", fmt.Errorf("unable to load private key: %v", err)
	}

	opts := jose.SignerOptions{NonceSource: nonceSrc}
	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.EdDSA,
		Key: jose.JSONWebKey{
			Key:   pk.GetKey(),
			KeyID: pk.KeyId(),
		}},
		&opts)
	if err != nil {
		return "", fmt.Errorf("unable to init signer: %v", err)
	}

	data, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	signedData, err := signer.Sign(data)
	if err != nil {
		return "", fmt.Errorf("unable to signe payload: %v", err)
	}

	jws, err := signedData.CompactSerialize()
	if err != nil {
		return "", fmt.Errorf("unable to serialize payload: %v", err)
	}

	return jws, nil
}
