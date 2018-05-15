package platform

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

// PlatformModel is a gorp wrapper around sdk.PlatformModel
type platformModel sdk.PlatformModel
type dbProjectPlatform sdk.ProjectPlatform

func init() {
	gorpmapping.Register(gorpmapping.New(platformModel{}, "platform_model", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbProjectPlatform{}, "project_platform", true, "id"))

}
