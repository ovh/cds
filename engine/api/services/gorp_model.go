package services

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
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
