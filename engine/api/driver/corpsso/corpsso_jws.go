package corpsso

import (
	"encoding/json"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gpg"

	jose "gopkg.in/square/go-jose.v2"
)

type ssoSigninToken struct {
	IssuedAt   int64                  `json:"iat"`
	RequireMFA bool                   `json:"requireMFA,omitempty"`
	Data       map[string]interface{} `json:"data"`
}

type nonceSource struct{}

func (n nonceSource) Nonce() (string, error) {
	return sdk.UUID(), nil
}

var nonceSrc jose.NonceSource = nonceSource{}

func prepareSSORequest(requestSigningKey string, request sdk.AuthSigninConsumerToken) (string, error) {
	pk, err := gpg.NewPrivateKeyFromPem(requestSigningKey, "")
	if err != nil {
		return "", sdk.WrapError(err, "unable to load private key")
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
		return "", sdk.WrapError(err, "unable to init signer")
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return "", sdk.WithStack(err)
	}
	var requestData map[string]interface{}
	if err := sdk.JSONUnmarshal(requestJSON, &requestData); err != nil {
		return "", sdk.WithStack(err)
	}

	data, err := json.Marshal(ssoSigninToken{
		Data:       requestData,
		IssuedAt:   request.IssuedAt,
		RequireMFA: request.RequireMFA,
	})
	if err != nil {
		return "", sdk.WithStack(err)
	}

	signedData, err := signer.Sign(data)
	if err != nil {
		return "", sdk.WrapError(err, "unable to signe payload")
	}

	jws, err := signedData.CompactSerialize()
	if err != nil {
		return "", sdk.WrapError(err, "unable to serialize payload")
	}

	return jws, nil
}
