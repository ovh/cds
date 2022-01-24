package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestLoadByNameAsAdmin(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	_ = event.Initialize(context.Background(), db.DbMap, cache, nil)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	test.NoError(t, application.Insert(db, *proj, &app))

	actual, err := application.LoadByName(db, key, "my-app")
	test.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadByNameAsUser(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	require.NoError(t, application.Insert(db, *proj, &app))

	_, _ = assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	actual, err := application.LoadByName(db, key, "my-app")
	assert.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadByIDAsAdmin(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	require.NoError(t, application.Insert(db, *proj, &app))

	actual, err := application.LoadByID(db, app.ID)
	require.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadByIDAsUser(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	key := sdk.RandomString(10)

	proj := assets.InsertTestProject(t, db, cache, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	require.NoError(t, application.Insert(db, *proj, &app))

	_, _ = assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	actual, err := application.LoadByID(db, app.ID)
	assert.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadAllAsAdmin(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	app := sdk.Application{
		Name: "my-app",
		Metadata: sdk.Metadata{
			"bla": "bla",
		},
	}

	app2 := sdk.Application{
		Name: "my-app2",
		Metadata: sdk.Metadata{
			"bla": "bla",
		},
	}

	require.NoError(t, application.Insert(db, *proj, &app))
	require.NoError(t, application.Insert(db, *proj, &app2))

	actual, err := application.LoadAll(db, proj.Key)
	require.NoError(t, err)

	assert.Equal(t, 2, len(actual))

	for _, a := range actual {
		assert.EqualValues(t, app.Metadata, a.Metadata)
		assert.EqualValues(t, app2.Metadata, a.Metadata)
	}
}

func TestLoadAllAsUser(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	app2 := sdk.Application{
		Name: "my-app2",
	}

	require.NoError(t, application.Insert(db, *proj, &app))
	require.NoError(t, application.Insert(db, *proj, &app2))

	_, _ = assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	actual, err := application.LoadAll(db, proj.Key)
	test.NoError(t, err)

	assert.Equal(t, 2, len(actual))
}

func TestLoadByWorkflowID(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	key := sdk.RandomString(10)

	proj := assets.InsertTestProject(t, db, cache, key, key)
	app := sdk.Application{
		Name:       "my-app",
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	require.NoError(t, application.Insert(db, *proj, &app))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	require.NoError(t, pipeline.InsertPipeline(db, &pip))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
			},
		},
	}

	test.NoError(t, workflow.RenameNode(context.TODO(), db, &w))

	proj, _ = project.LoadByID(db, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	require.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	actuals, err := application.LoadByWorkflowID(db, w.ID)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(actuals))
	assert.Equal(t, app.Name, actuals[0].Name)
	assert.Equal(t, proj.ID, actuals[0].ProjectID)

}

func TestWithRepositoryStrategy(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	key := sdk.RandomString(10)

	proj := assets.InsertTestProject(t, db, cache, key, key)
	app := &sdk.Application{
		Name:       "my-app",
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	require.NoError(t, application.Insert(db, *proj, app))

	app.RepositoryStrategy = sdk.RepositoryStrategy{
		Branch:         "{{.git.branch}}",
		ConnectionType: "https",
		DefaultBranch:  "master",
		User:           "user",
		Password:       "password",
		SSHKeyContent:  "content",
	}

	require.NoError(t, application.Update(db, app))
	require.Equal(t, "user", app.RepositoryStrategy.User)
	require.Equal(t, sdk.PasswordPlaceholder, app.RepositoryStrategy.Password)
	require.Equal(t, "", app.RepositoryStrategy.SSHKeyContent) // it depends on the connection type

	var err error

	app, err = application.LoadByID(db, app.ID)
	require.NoError(t, err)
	app.RepositoryStrategy.Password = "password2"
	require.NoError(t, application.Update(db, app))

	app, err = application.LoadByIDWithClearVCSStrategyPassword(db, app.ID)
	require.NoError(t, err)
	require.Equal(t, "user", app.RepositoryStrategy.User)
	require.Equal(t, "password2", app.RepositoryStrategy.Password)
	require.Equal(t, "", app.RepositoryStrategy.SSHKeyContent) // it depends on the connection type

	app, err = application.LoadByID(db, app.ID)
	require.NoError(t, err)
	require.Equal(t, "user", app.RepositoryStrategy.User)
	require.Equal(t, sdk.PasswordPlaceholder, app.RepositoryStrategy.Password)
	require.Equal(t, "", app.RepositoryStrategy.SSHKeyContent) // it depends on the connection type

	app.RepositoryStrategy.ConnectionType = "ssh"
	app.RepositoryStrategy.SSHKeyContent = "ssh_key"
	app.RepositoryStrategy.SSHKey = "ssh_key"

	require.NoError(t, application.Update(db, app))
	require.Equal(t, "user", app.RepositoryStrategy.User)
	require.Equal(t, sdk.PasswordPlaceholder, app.RepositoryStrategy.Password)
	require.Equal(t, "", app.RepositoryStrategy.SSHKeyContent) // it depends on the connection type

}

func Test_LoadAllVCStrategyAllApps(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	key := sdk.RandomString(10)

	proj := assets.InsertTestProject(t, db, cache, key, key)
	app1 := &sdk.Application{Name: "my-app1", ProjectKey: proj.Key, ProjectID: proj.ID, RepositoryStrategy: sdk.RepositoryStrategy{
		Password: "secret1",
	}}
	app2 := &sdk.Application{Name: "my-app2", ProjectKey: proj.Key, ProjectID: proj.ID, RepositoryStrategy: sdk.RepositoryStrategy{
		Password: "secret2",
	}}
	require.NoError(t, application.Insert(db, *proj, app1))
	require.NoError(t, application.Insert(db, *proj, app2))

	apps, err := application.LoadAllByIDsWithDecryption(db, []int64{app1.ID, app2.ID})
	require.NoError(t, err)

	require.Len(t, apps, 2)
	app1Check := false
	app2Check := false
	for _, app := range apps {
		switch app.Name {
		case "my-app1":
			app1Check = true
			require.Equal(t, "secret1", app.RepositoryStrategy.Password)
		case "my-app2":
			app2Check = true
			require.Equal(t, "secret2", app.RepositoryStrategy.Password)
		default:
			t.Fail()
		}
	}
	require.True(t, app1Check)
	require.True(t, app2Check)
}
