package accesstoken

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type accessToken sdk.AccessToken

func init() {
	gorpmapping.Register(
		gorpmapping.New(accessToken{}, "access_token", false, "id"),
	)
}
