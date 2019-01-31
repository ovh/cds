package integration

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

// IntegrationModel is a gorp wrapper around sdk.IntegrationModel
type integrationModel sdk.IntegrationModel
type dbProjectIntegration sdk.ProjectIntegration

func init() {
	gorpmapping.Register(gorpmapping.New(integrationModel{}, "integration_model", true, "id"))
	gorpmapping.Register(gorpmapping.New(dbProjectIntegration{}, "project_integration", true, "id"))

}
