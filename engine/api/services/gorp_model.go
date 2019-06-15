package services

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type service struct {
	sdk.Service
	gorpmapping.SignedEntity
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(service{}, "services", true, "id"),
	)
}
