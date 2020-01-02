package services

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type service struct {
	sdk.Service
	gorpmapping.SignedEntity
}

func (s service) Canonical() gorpmapping.CanonicalForms {
	return []gorpmapping.CanonicalForm{
		"{{.ID}}{{.Name}}{{.Type}}",
	}
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(service{}, "service", true, "id"),
	)
}
