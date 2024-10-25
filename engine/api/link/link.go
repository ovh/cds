package link

import (
	"context"
	"github.com/ovh/cds/sdk"
)

type LinkDriver interface {
	GetUserInfo(context.Context, sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error)
	GetDriver() sdk.Driver
}
