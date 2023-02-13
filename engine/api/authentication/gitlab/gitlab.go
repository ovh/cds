package gitlab

import (
	"context"
	"github.com/ovh/cds/engine/api/driver/gitlab"
	"time"

	"github.com/ovh/cds/sdk"
)

// NewDriver returns a new Gitlab auth driver for given config.
func NewDriver(signupDisabled bool, cdsURL, url, applicationID, secret, orga string) sdk.AuthDriver {
	return &authDriver{
		signupDisabled: signupDisabled,
		organization:   orga,
		driver:         gitlab.NewGitlabDriver(cdsURL, url, applicationID, secret),
	}
}

type authDriver struct {
	signupDisabled bool
	driver         sdk.Driver
	organization   string
}

func (d *authDriver) GetDriver() sdk.Driver {
	return d.driver
}

func (d *authDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerGitlab,
		SignupDisabled: d.signupDisabled,
	}
}

func (d *authDriver) GetSessionDuration() time.Duration {
	return time.Hour * 24 * 30 // 1 month session
}

func (d *authDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	info, err := d.driver.GetUserInfoFromDriver(ctx, req)
	if err != nil {
		return info, err
	}
	info.Organization = d.organization
	return info, nil
}
