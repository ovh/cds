package ssh

import (
	"fmt"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

const SecretOTPType = "secret_otp_type"

func secretOTP(b *backend) *framework.Secret {
	return &framework.Secret{
		Type: SecretOTPType,
		Fields: map[string]*framework.FieldSchema{
			"otp": &framework.FieldSchema{
				Type:        framework.TypeString,
				Description: "One time password",
			},
		},

		Revoke: b.secretOTPRevoke,
	}
}

func (b *backend) secretOTPRevoke(req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	otpRaw, ok := req.Secret.InternalData["otp"]
	if !ok {
		return nil, fmt.Errorf("secret is missing internal data")
	}
	otp, ok := otpRaw.(string)
	if !ok {
		return nil, fmt.Errorf("secret is missing internal data")
	}

	err := req.Storage.Delete("otp/" + b.salt.SaltID(otp))
	if err != nil {
		return nil, err
	}
	return nil, nil
}
