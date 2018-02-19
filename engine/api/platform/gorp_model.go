package platform

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

// PlatformModel is a gorp wrapper around sdk.PlatformModel
type PlatformModel sdk.PlatformModel

func init() {
	gorpmapping.Register(gorpmapping.New(PlatformModel{}, "platform_model", true, "id"))
}
