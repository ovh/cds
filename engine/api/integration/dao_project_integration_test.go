package integration_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test/assets"

	"github.com/ovh/cds/engine/api/integration"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCRUDIntegration(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

	project.Delete(db, "key")

	proj := sdk.Project{
		Name: "test proj",
		Key:  "key",
	}
	assert.NoError(t, project.Insert(db, &proj))

	model, err := integration.LoadModelByNameWithClearPassword(context.TODO(), db, sdk.KafkaIntegration.Name)
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

	reloadedInteg, err := integration.LoadIntegrationsByProjectID(context.TODO(), db, proj.ID)
	t.Logf("%+v", reloadedInteg)
	require.NoError(t, err)
	require.Len(t, reloadedInteg, 1)
	assert.Equal(t, sdk.PasswordPlaceholder, reloadedInteg[0].Config["password"].Value)

	reloadedInteg, err = integration.LoadIntegrationsByProjectIDWithClearPassword(context.TODO(), db, proj.ID)
	require.NoError(t, err)
	require.Len(t, reloadedInteg, 1)
	assert.Equal(t, "mypassword", reloadedInteg[0].Config["password"].Value)

	require.NoError(t, integration.DeleteIntegration(db, reloadedInteg[0]))
}

func TestLoadAllIntegrationForAllProject(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitializeDB)

	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, cache, key1, key1)

	key2 := sdk.RandomString(10)
	proj2 := assets.InsertTestProject(t, db, cache, key2, key2)

	model, err := integration.LoadModelByNameWithClearPassword(context.TODO(), db, sdk.KafkaIntegration.Name)
	require.NoError(t, err)

	integ1 := sdk.ProjectIntegration{
		Config: map[string]sdk.IntegrationConfigValue{
			"token": {
				Value: "secret1",
				Type:  sdk.IntegrationConfigTypePassword,
			},
		},
		IntegrationModelID: model.ID,
		Name:               model.Name,
		ProjectID:          proj1.ID,
	}
	integ2 := sdk.ProjectIntegration{
		Config: map[string]sdk.IntegrationConfigValue{
			"token": {
				Value: "secret2",
				Type:  sdk.IntegrationConfigTypePassword,
			},
		},
		IntegrationModelID: model.ID,
		Name:               model.Name,
		ProjectID:          proj2.ID,
	}

	require.NoError(t, integration.InsertIntegration(db, &integ1))
	require.NoError(t, integration.InsertIntegration(db, &integ2))

	ints, err := integration.LoadAllIntegrationsForProjectsWithDecryption(context.TODO(), db, []int64{proj1.ID, proj2.ID})
	require.NoError(t, err)
	require.Len(t, ints, 2)

	pp1s := ints[proj1.ID]
	require.Len(t, pp1s, 1)
	require.Equal(t, "secret1", pp1s[0].Config["token"].Value)

	pp2s := ints[proj2.ID]
	require.Len(t, pp2s, 1)
	require.Equal(t, "secret2", pp2s[0].Config["token"].Value)
}
