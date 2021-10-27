package workerhook_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/workerhook"
	"github.com/ovh/cds/sdk"
)

func TestCRUD(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)
	project.Delete(db, "key")

	proj := sdk.Project{
		Name: "test proj",
		Key:  "key",
	}
	require.NoError(t, project.Insert(db, &proj))

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

	var h = sdk.WorkerHookProjectIntegrationModel{
		ProjectIntegrationID: integ.ID,
		Configuration: sdk.WorkerHookSetupTeardownConfig{
			ByCapabilities: map[string]sdk.WorkerHookSetupTeardownScripts{
				"docker": {
					Priority: 1,
					Setup:    "docker login -u {{.cds.integration.artifactory.username}} -p {{.cds.integration.artifactory.password}} {{.cds.integration.artifactory.docker.registry}}",
				},
			},
		},
	}

	require.NoError(t, workerhook.Insert(context.TODO(), db, &h))
	require.NoError(t, workerhook.Update(context.TODO(), db, &h))

	_, err = workerhook.LoadByProjectIntegrationID(context.TODO(), db, h.ProjectIntegrationID)
	require.NoError(t, err)

	_, err = workerhook.LoadEnabledByProjectIntegrationID(context.TODO(), db, h.ProjectIntegrationID)
	require.NoError(t, err)

	_, err = workerhook.LoadAll(context.TODO(), db)
	require.NoError(t, err)

	_, err = workerhook.LoadByID(context.TODO(), db, h.ID)
	require.NoError(t, err)

	require.NoError(t, project.Delete(db, "key"))
}
