package pipeline_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestInsertPipeline(t *testing.T) {
	db, cache := test.SetupPG(t)
	pk := sdk.RandomString(8)

	p := sdk.Project{
		Key:  pk,
		Name: pk,
	}
	if err := project.Insert(db, cache, &p, nil); err != nil {
		t.Fatalf("Cannot insert project : %s", err)
	}

	tests := []struct {
		name    string
		p       *sdk.Pipeline
		wantErr bool
	}{
		{
			name:    "InsertPipeline should fail with sdk.ErrInvalidName",
			p:       &sdk.Pipeline{},
			wantErr: true,
		},
		{
			name: "InsertPipeline should fail with sdk.ErrInvalidType",
			p: &sdk.Pipeline{
				Name: "Name",
			},
			wantErr: true,
		},
		{
			name: "InsertPipeline should fail with sdk.ErrInvalidProject",
			p: &sdk.Pipeline{
				Name: "Name",
				Type: sdk.DeploymentPipeline,
			},
			wantErr: true,
		},
		{
			name: "InsertPipeline should not fail",
			p: &sdk.Pipeline{
				Name:      "Name",
				Type:      sdk.DeploymentPipeline,
				ProjectID: p.ID,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		if err := pipeline.InsertPipeline(db, cache, &p, tt.p, nil); (err != nil) != tt.wantErr {
			t.Errorf("%q. InsertPipeline() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestInsertPipelineWithParemeters(t *testing.T) {
	db, cache := test.SetupPG(t)
	pk := sdk.RandomString(8)

	p := sdk.Project{
		Key:  pk,
		Name: pk,
	}
	if err := project.Insert(db, cache, &p, nil); err != nil {
		t.Fatalf("Cannot insert project : %s", err)
	}

	pip := &sdk.Pipeline{
		Name:      "Name",
		Type:      sdk.DeploymentPipeline,
		ProjectID: p.ID,
		Parameter: []sdk.Parameter{
			{
				Name:  "P1",
				Value: "V1",
				Type:  sdk.StringParameter,
			},
			{
				Name:  "P2",
				Value: "V2",
				Type:  sdk.StringParameter,
			},
		},
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, &p, pip, nil))

	pip1, err := pipeline.LoadPipeline(db, p.Key, "Name", true)
	test.NoError(t, err)

	assert.Equal(t, len(pip.Parameter), len(pip1.Parameter))
}

func TestInsertPipelineWithWithWrongParemeters(t *testing.T) {
	db, cache := test.SetupPG(t)
	pk := sdk.RandomString(8)

	p := sdk.Project{
		Key:  pk,
		Name: pk,
	}
	if err := project.Insert(db, cache, &p, nil); err != nil {
		t.Fatalf("Cannot insert project : %s", err)
	}

	pip := &sdk.Pipeline{
		Name:      "Name",
		Type:      sdk.DeploymentPipeline,
		ProjectID: p.ID,
		Parameter: []sdk.Parameter{
			{
				Value: "V1",
				Type:  sdk.StringParameter,
			},
			{
				Name:  "P2 2",
				Value: "V2",
				Type:  sdk.StringParameter,
			},
		},
	}
	assert.Error(t, pipeline.InsertPipeline(db, cache, &p, pip, nil))
}

func TestLoadByWorkflowID(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
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
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
			Context: &sdk.WorkflowNodeContext{
				Application: &app,
			},
		},
	}

	proj, _ = project.LoadByID(db, cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	test.NoError(t, workflow.Insert(db, cache, &w, proj, u))

	actuals, err := pipeline.LoadByWorkflowID(db, w.ID)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(actuals))
	assert.Equal(t, pip.ID, actuals[0].ID)
}
