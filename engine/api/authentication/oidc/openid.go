package oidc

import (
	"context"
	"github.com/ovh/cds/engine/api/driver/openid"
	"time"

	"github.com/ovh/cds/sdk"
)

// NewDriver returns a new OIDC auth driver for given config.
func NewDriver(signupDisabled bool, cdsURL, url, clientID, clientSecret, orga string) (sdk.AuthDriver, error) {
	driv, err := openid.NewOpenIDDriver(cdsURL, url, clientID, clientSecret)
	if err != nil {
		return nil, err
	}

	return &authDriver{
		signupDisabled: signupDisabled,
		driver:         driv,
		organization:   orga,
	}, nil
}

type authDriver struct {
	signupDisabled bool
	organization   string
	driver         sdk.Driver
}

func (d authDriver) GetDriver() sdk.Driver {
	return d.driver
}

func (d authDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerOIDC,
		SignupDisabled: d.signupDisabled,
	}
}

func (d authDriver) GetSessionDuration() time.Duration {
	return time.Hour * 24 * 30 // 1 month session
}

func (d authDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	info, err := d.driver.GetUserInfoFromDriver(ctx, req)
	if err != nil {
		return info, err
	}
	info.Organization = d.organization
	return info, nil
}
