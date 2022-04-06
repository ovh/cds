package application_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/integration"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_LoadAllDeploymentAllApps(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	app1 := sdk.Application{
		Name: "my-app1",
	}
	app2 := sdk.Application{
		Name: "my-app2",
	}
	require.NoError(t, application.Insert(db, *proj, &app1))
	require.NoError(t, application.Insert(db, *proj, &app2))

	pfname := sdk.RandomString(10)
	pf := sdk.IntegrationModel{
		Name:       pfname,
		Deployment: true,
		AdditionalDefaultConfig: sdk.IntegrationConfig{
			"token": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypePassword,
				Value: "my-secret-token",
			},
		},
	}
	test.NoError(t, integration.InsertModel(db, &pf))
	defer func() { _ = integration.DeleteModel(context.TODO(), db, pf.ID) }()

	pp := sdk.ProjectIntegration{
		Model:              pf,
		Name:               pf.Name,
		IntegrationModelID: pf.ID,
		ProjectID:          proj.ID,
	}
	test.NoError(t, integration.InsertIntegration(db, &pp))

	cfg1 := sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "secret1",
		},
	}
	cfg2 := sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "secret2",
		},
	}
	require.NoError(t, application.SetDeploymentStrategy(db, proj.ID, app1.ID, pf.ID, pfname, cfg1))
	require.NoError(t, application.SetDeploymentStrategy(db, proj.ID, app2.ID, pf.ID, pfname, cfg2))

	deps, err := application.LoadAllDeploymnentForAppsWithDecryption(context.TODO(), db, []int64{app1.ID, app2.ID})
	require.NoError(t, err)

	require.Len(t, deps, 2)
	require.NotNil(t, deps[app1.ID])
	require.NotNil(t, deps[app2.ID])
	require.Len(t, deps[app1.ID], 1)
	require.Len(t, deps[app2.ID], 1)
	require.Equal(t, "secret1", deps[app1.ID][pp.ID]["token"].Value)
	require.Equal(t, "secret2", deps[app2.ID][pp.ID]["token"].Value)

	pf.AdditionalDefaultConfig = sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "my-secret-token",
		},
		"bar": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypeString,
			Value: "foo",
		},
	}

	test.NoError(t, integration.UpdateModel(context.TODO(), db, &pf))

	st, err := application.LoadDeploymentStrategies(context.TODO(), db, app1.ID, false)
	require.NoError(t, err)
	require.Len(t, st, 1)
	require.Equal(t, st[pf.Name]["token"].Value, "**********")
	require.Equal(t, st[pf.Name]["bar"].Value, "foo")
}
