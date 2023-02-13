package ldap

import (
	"context"
	"github.com/ovh/cds/engine/api/driver/ldap"
	"time"

	"github.com/ovh/cds/sdk"
)

var _ sdk.AuthDriver = new(AuthDriver)

type AuthDriver struct {
	signupDisabled bool
	driver         sdk.Driver
}

// NewDriver returns a new ldap auth driver.
func NewDriver(ctx context.Context, signupDisabled bool, cfg ldap.Config) (sdk.AuthDriver, error) {
	var d = AuthDriver{
		signupDisabled: signupDisabled,
	}

	ldap, err := ldap.NewLdapDriver(ctx, cfg)
	if err != nil {
		return d, err
	}
	d.driver = ldap

	return d, nil
}

func (d AuthDriver) GetDriver() sdk.Driver {
	return d.driver
}

func (d AuthDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerLDAP,
		SignupDisabled: d.signupDisabled,
	}
}

func (d AuthDriver) GetSessionDuration() time.Duration {
	return time.Hour * 24 * 30 // 1 month session
}

func (d AuthDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	return d.driver.GetUserInfoFromDriver(ctx, req)
}
