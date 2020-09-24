package services

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type service struct {
	sdk.Service
	gorpmapper.SignedEntity
}

func (s service) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Name}}{{.Type}}",
	}
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(service{}, "service", true, "id"),
		gorpmapping.New(sdk.ServiceStatus{}, "service_status", true, "id"),
	)
}
