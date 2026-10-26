package builtin

import (
	"context"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

var _ sdk.Driver = new(BuiltinDriver)

// NewBuiltinDriver returns a new initialized driver for builtin authentication.
func NewBuiltinDriver() sdk.Driver {
	return &BuiltinDriver{}
}

type BuiltinDriver struct{}

func (d BuiltinDriver) GetUserInfoFromDriver(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var userInfo sdk.AuthDriverUserInfo

	token, err := req.StringE("token")
	if err != nil {
		return userInfo, sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid authentication token")
	}

	consumerID, _, err := CheckSigninConsumerToken(token)
	if err != nil {
		return userInfo, err
	}

	log.Debug(ctx, "builtin.GetUserInfo> %s", consumerID)

	return sdk.AuthDriverUserInfo{
		ExternalID: consumerID,
	}, nil
}

// CheckSigninRequest checks that given driver request is valid for a signin with auth builtin.
func (d BuiltinDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	token, err := req.StringE("token")
	if err != nil {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid authentication token")
	}

	_, _, err = CheckSigninConsumerToken(token)
	return err
}
