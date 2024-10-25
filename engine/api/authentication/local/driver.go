package local

import (
	"context"
	"github.com/ovh/cds/engine/api/driver/local"
	"time"

	"github.com/ovh/cds/sdk"
)

var _ sdk.AuthDriver = new(AuthDriver)

// NewDriver returns a new initialized driver for local authentication.
func NewDriver(ctx context.Context, signupDisabled bool, uiURL, allowedDomains string, orga string) sdk.AuthDriver {
	return &AuthDriver{
		signupDisabled: signupDisabled,
		organization:   orga,
		driver:         local.NewLocalDriver(ctx, allowedDomains),
	}
}

// AuthDriver for local authentication.
type AuthDriver struct {
	signupDisabled bool
	organization   string
	driver         sdk.Driver
}

func (d AuthDriver) GetDriver() sdk.Driver {
	return d.driver
}

// GetManifest .
func (d AuthDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerLocal,
		SignupDisabled: d.signupDisabled,
	}
}

// GetSessionDuration .
func (d AuthDriver) GetSessionDuration() time.Duration {
	return time.Hour * 24 * 30 // 1 month session
}

// GetUserInfo .
func (d AuthDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	user, _ := d.driver.GetUserInfoFromDriver(ctx, req)
	user.Organization = d.organization
	return user, nil
}
