package application_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	event.Initialize(event.KafkaConfig{}, cache)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, nil)
	app := sdk.Application{
		Name: "my-app",
	}

	test.NoError(t, application.Insert(db, cache, proj, &app, nil))

	actual, err := application.LoadByName(db, cache, key, "my-app")
	test.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadByNameAsUser(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, nil)
	app := sdk.Application{
		Name: "my-app",
	}

	test.NoError(t, application.Insert(db, cache, proj, &app, nil))

	_, _ = assets.InsertLambdaUser(db, &proj.ProjectGroups[0].Group)

	actual, err := application.LoadByName(db, cache, key, "my-app")
	assert.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadByIDAsAdmin(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, nil)
	app := sdk.Application{
		Name: "my-app",
	}

	test.NoError(t, application.Insert(db, cache, proj, &app, nil))

	actual, err := application.LoadByID(db, cache, app.ID)
	test.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadByIDAsUser(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)

	proj := assets.InsertTestProject(t, db, cache, key, key, nil)
	app := sdk.Application{
		Name: "my-app",
	}

	test.NoError(t, application.Insert(db, cache, proj, &app, nil))

	_, _ = assets.InsertLambdaUser(db, &proj.ProjectGroups[0].Group)

	actual, err := application.LoadByID(db, cache, app.ID)
	assert.NoError(t, err)

	assert.Equal(t, app.Name, actual.Name)
	assert.Equal(t, proj.ID, actual.ProjectID)
	assert.Equal(t, proj.Key, actual.ProjectKey)
}

func TestLoadAllAsAdmin(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, nil)
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

	test.NoError(t, application.Insert(db, cache, proj, &app, nil))
	test.NoError(t, application.Insert(db, cache, proj, &app2, nil))

	actual, err := application.LoadAll(db, cache, proj.Key)
	test.NoError(t, err)

	assert.Equal(t, 2, len(actual))

	for _, a := range actual {
		assert.EqualValues(t, app.Metadata, a.Metadata)
		assert.EqualValues(t, app2.Metadata, a.Metadata)
	}
}

func TestLoadAllAsUser(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, nil)
	app := sdk.Application{
		Name: "my-app",
	}

	app2 := sdk.Application{
		Name: "my-app2",
	}

	test.NoError(t, application.Insert(db, cache, proj, &app, nil))
	test.NoError(t, application.Insert(db, cache, proj, &app2, nil))

	_, _ = assets.InsertLambdaUser(db, &proj.ProjectGroups[0].Group)

	actual, err := application.LoadAll(db, cache, proj.Key)
	test.NoError(t, err)

	assert.Equal(t, 2, len(actual))
}

func TestLoadByWorkflowID(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)

	proj := assets.InsertTestProject(t, db, cache, key, key, nil)
	app := sdk.Application{
		Name:       "my-app",
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, application.Insert(db, cache, proj, &app, u))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
			},
		},
	}

	test.NoError(t, workflow.RenameNode(db, &w))
	(&w).RetroMigrate()

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	actuals, err := application.LoadByWorkflowID(db, w.ID)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(actuals))
	assert.Equal(t, app.Name, actuals[0].Name)
	assert.Equal(t, proj.ID, actuals[0].ProjectID)

}
