package integration_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/integration"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCRUDIntegration(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	project.Delete(db, "key")

	proj := sdk.Project{
		Name: "test proj",
		Key:  "key",
	}
	assert.NoError(t, project.Insert(db, &proj))

	model, err := integration.LoadModelByName(db, sdk.KafkaIntegration.Name, false)
	require.NoError(t, err)

	integ := sdk.ProjectIntegration{
		Config:             model.DefaultConfig.Clone(),
		IntegrationModelID: model.ID,
		Name:               model.Name,
		ProjectID:          proj.ID,
	}
	pass := integ.Config["password"]
	pass.Value = "mypassword"
	integ.Config["password"] = pass
	require.NoError(t, integration.InsertIntegration(db, &integ))
	assert.Equal(t, sdk.PasswordPlaceholder, integ.Config["password"].Value)

	reloadedInteg, err := integration.LoadIntegrationsByProjectID(db, proj.ID, false)
	t.Logf("%+v", reloadedInteg)
	require.NoError(t, err)
	require.Len(t, reloadedInteg, 1)
	assert.Equal(t, sdk.PasswordPlaceholder, reloadedInteg[0].Config["password"].Value)

	reloadedInteg, err = integration.LoadIntegrationsByProjectID(db, proj.ID, true)
	require.NoError(t, err)
	require.Len(t, reloadedInteg, 1)
	assert.Equal(t, "mypassword", reloadedInteg[0].Config["password"].Value)

	require.NoError(t, integration.DeleteIntegration(db, reloadedInteg[0]))

}
