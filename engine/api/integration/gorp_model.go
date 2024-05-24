package integration

import (
	"context"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// IntegrationModel is a gorp wrapper around sdk.IntegrationModel
type integrationModel struct {
	gorpmapper.SignedEntity
	sdk.IntegrationModel
}

func (e integrationModel) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{e.Name}
	return gorpmapper.CanonicalForms{
		"{{.Name}}",
	}
}

type integrationModelSlice []integrationModel

func (s integrationModelSlice) IntegrationModel() []sdk.IntegrationModel {
	var integrations = make([]sdk.IntegrationModel, len(s))
	for i, p := range s {
		isValid, err := gorpmapping.CheckSignature(p, p.Signature)
		if err != nil {
			return nil
		}
		if !isValid {
			log.Error(context.Background(), "integration.IntegrationModel> model %d data corrupted", p.ID)
			continue
		}
		integrations[i] = p.IntegrationModel
	}
	return integrations
}

type dbProjectIntegration struct {
	gorpmapper.SignedEntity
	sdk.ProjectIntegration
}

func (e dbProjectIntegration) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{e.IntegrationModelID, e.ProjectID, e.Config}
	return gorpmapper.CanonicalForms{
		"{{.IntegrationModelID}}{{.ProjectID}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(integrationModel{}, "integration_model", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbProjectIntegration{}, "project_integration", true, "id"))
}
