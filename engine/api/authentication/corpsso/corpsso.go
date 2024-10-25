package corpsso

import (
	"context"
	"github.com/ovh/cds/engine/api/driver/corpsso"
	"time"

	"github.com/ovh/cds/sdk"
)

type authDriver struct {
	driver sdk.Driver
}

func NewDriver(cfg corpsso.SSOConfig) sdk.AuthDriver {
	var d = authDriver{corpsso.NewCorpSSODriver(cfg)}
	return d
}

func (d authDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerCorporateSSO,
		SignupDisabled: false,
		SupportMFA:     d.driver.(corpsso.CorpSSODriver).Config.MFASupportEnabled,
	}
}

func (d authDriver) GetDriver() sdk.Driver {
	return d.driver
}

func (d authDriver) GetSessionDuration() time.Duration {
	return 24 * time.Hour
}

func (d authDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	return d.driver.GetUserInfoFromDriver(ctx, req)
}
