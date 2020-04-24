package integration

import (
	"context"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk/log"

	"github.com/ovh/cds/sdk"
)

// IntegrationModel is a gorp wrapper around sdk.IntegrationModel
type integrationModel struct {
	gorpmapping.SignedEntity
	sdk.IntegrationModel
}

func (e integrationModel) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{e.Name}
	return gorpmapping.CanonicalForms{
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
	gorpmapping.SignedEntity
	sdk.ProjectIntegration
}

func (e dbProjectIntegration) Canonical() gorpmapping.CanonicalForms {
	var _ = []interface{}{e.IntegrationModelID, e.ProjectID}
	return gorpmapping.CanonicalForms{
		"{{.IntegrationModelID}}{{.ProjectID}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(integrationModel{}, "integration_model", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbProjectIntegration{}, "project_integration", true, "id"))
}
