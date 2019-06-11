package accesstoken

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(
		gorpmapping.New(sdk.AuthSession{}, "auth_session", false, "id"),
		gorpmapping.New(sdk.AuthConsumer{}, "auth_consumer", false, "id"),
	)
}
