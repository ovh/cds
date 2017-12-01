package services

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type service sdk.Service

func init() {
	gorpmapping.Register(
		gorpmapping.New(service{}, "services", false, "name"),
	)
}
