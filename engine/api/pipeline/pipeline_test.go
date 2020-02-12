package pipeline_test

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestInsertPipeline(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	pk := sdk.RandomString(8)

	p := sdk.Project{
		Key:  pk,
		Name: pk,
	}
	if err := project.Insert(db, cache, &p); err != nil {
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
			},
			wantErr: true,
		},
		{
			name: "InsertPipeline should not fail",
			p: &sdk.Pipeline{
				Name:      "Name",
				ProjectID: p.ID,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		if err := pipeline.InsertPipeline(db, cache, &p, tt.p); (err != nil) != tt.wantErr {
			t.Errorf("%q. InsertPipeline() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestInsertPipelineWithParemeters(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	pk := sdk.RandomString(8)

	p := sdk.Project{
		Key:  pk,
		Name: pk,
	}
	if err := project.Insert(db, cache, &p); err != nil {
		t.Fatalf("Cannot insert project : %s", err)
	}

	pip := &sdk.Pipeline{
		Name:      "Name",
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

	test.NoError(t, pipeline.InsertPipeline(db, cache, &p, pip))

	pip1, err := pipeline.LoadPipeline(context.TODO(), db, p.Key, "Name", true)
	test.NoError(t, err)

	assert.Equal(t, len(pip.Parameter), len(pip1.Parameter))
}

func TestInsertPipelineWithWithWrongParemeters(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	pk := sdk.RandomString(8)

	p := sdk.Project{
		Key:  pk,
		Name: pk,
	}
	if err := project.Insert(db, cache, &p); err != nil {
		t.Fatalf("Cannot insert project : %s", err)
	}

	pip := &sdk.Pipeline{
		Name:      "Name",
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
	assert.Error(t, pipeline.InsertPipeline(db, cache, &p, pip))
}

func TestLoadByWorkflowID(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)

	proj := assets.InsertTestProject(t, db, cache, key, key)
	app := sdk.Application{
		Name:       "my-app",
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, application.Insert(db, cache, proj, &app))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

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

	test.NoError(t, workflow.RenameNode(context.TODO(), db, &w))

	proj, _ = project.LoadByID(db, cache, proj.ID, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, &w, proj))

	actuals, err := pipeline.LoadByWorkflowID(db, w.ID)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(actuals))
	assert.Equal(t, pip.ID, actuals[0].ID)
}

func TestLoadByWorkerModel(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	g1 := group.SharedInfraGroup
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	model1 := sdk.Model{Name: sdk.RandomString(10), Group: g1, GroupID: g1.ID}
	model2 := sdk.Model{Name: sdk.RandomString(10), Group: g2, GroupID: g2.ID}

	projectKey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, projectKey, projectKey)

	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: proj.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	// first pipeline with requirement shared.infra/model
	pip1 := sdk.Pipeline{ProjectID: proj.ID, ProjectKey: proj.Key, Name: sdk.RandomString(10)}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip1))
	job1 := sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Requirements: []sdk.Requirement{{
				Type:  sdk.ModelRequirement,
				Name:  fmt.Sprintf("%s/%s --privileged", model1.Group.Name, model1.Name),
				Value: fmt.Sprintf("%s/%s --privileged", model1.Group.Name, model1.Name),
			}},
		},
	}
	test.NoError(t, pipeline.InsertJob(db, &job1, 0, &pip1))

	// second pipeline with requirement model
	pip2 := sdk.Pipeline{ProjectID: proj.ID, ProjectKey: proj.Key, Name: sdk.RandomString(10)}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip2))
	job2 := sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Requirements: []sdk.Requirement{{
				Type:  sdk.ModelRequirement,
				Name:  fmt.Sprintf("%s --privileged", model1.Name),
				Value: fmt.Sprintf("%s --privileged", model1.Name),
			}},
		},
	}
	test.NoError(t, pipeline.InsertJob(db, &job2, 0, &pip2))

	// third pipeline with requirement group/model
	pip3 := sdk.Pipeline{ProjectID: proj.ID, ProjectKey: proj.Key, Name: sdk.RandomString(10)}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip3))
	job3 := sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Requirements: []sdk.Requirement{{
				Type:  sdk.ModelRequirement,
				Name:  fmt.Sprintf("%s/%s --privileged", model2.Group.Name, model2.Name),
				Value: fmt.Sprintf("%s/%s --privileged", model2.Group.Name, model2.Name),
			}},
		},
	}
	test.NoError(t, pipeline.InsertJob(db, &job3, 0, &pip3))

	pips, err := pipeline.LoadByWorkerModel(context.TODO(), db, &model1)
	assert.NoError(t, err)
	if !assert.Equal(t, 2, len(pips)) {
		t.FailNow()
	}
	sort.Slice(pips, func(i, j int) bool { return pips[i].ID < pips[j].ID })
	assert.Equal(t, pip1.Name, pips[0].Name)
	assert.Equal(t, pip2.Name, pips[1].Name)

	pips, err = pipeline.LoadByWorkerModel(context.TODO(), db, &model2)
	assert.NoError(t, err)

	if !assert.Equal(t, 1, len(pips)) {
		t.FailNow()
	}
	assert.Equal(t, pip3.Name, pips[0].Name)

	pips, err = pipeline.LoadByWorkerModelAndGroupIDs(context.TODO(), db, &model1, []int64{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pips))

	pips, err = pipeline.LoadByWorkerModelAndGroupIDs(context.TODO(), db, &model1, sdk.Groups{*g2}.ToIDs())
	assert.NoError(t, err)
	if !assert.Equal(t, 2, len(pips)) {
		t.FailNow()
	}
	sort.Slice(pips, func(i, j int) bool { return pips[i].ID < pips[j].ID })
	assert.Equal(t, pip1.Name, pips[0].Name)
	assert.Equal(t, pip2.Name, pips[1].Name)

	pips, err = pipeline.LoadByWorkerModelAndGroupIDs(context.TODO(), db, &model2, sdk.Groups{*g2}.ToIDs())
	assert.NoError(t, err)

	if !assert.Equal(t, 1, len(pips)) {
		t.FailNow()
	}
	assert.Equal(t, pip3.Name, pips[0].Name)
}
