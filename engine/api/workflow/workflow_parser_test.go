package workflow_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func TestParseAndImport(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	//Pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pipeline",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	//Application
	repofullname := sdk.RandomString(10) + "/" + sdk.RandomString(10)
	app := &sdk.Application{
		Name:               sdk.RandomString(10),
		RepositoryFullname: repofullname,
		VCSServer:          "github",
	}
	test.NoError(t, application.Insert(db, cache, proj, app, u))

	//Environment
	envName := sdk.RandomString(10)
	env := &sdk.Environment{
		ProjectID: proj.ID,
		Name:      envName,
	}
	test.NoError(t, environment.InsertEnvironment(db, env))

	//Reload project
	proj, _ = project.Load(db, cache, proj.Key, u, project.LoadOptions.WithApplications, project.LoadOptions.WithEnvironments, project.LoadOptions.WithPipelines)

	input := &exportentities.Workflow{
		Name: "test-1",
		Workflow: map[string]exportentities.NodeEntry{
			"root": {
				PipelineName:    "pipeline",
				ApplicationName: app.Name,
			},
			"first": {
				PipelineName: "pipeline",
				DependsOn:    []string{"root"},
			},
			"second": {
				PipelineName: "pipeline",
				DependsOn:    []string{"first"},
			},
			"fork": {
				DependsOn: []string{"root"},
			},
			"third": {
				PipelineName: "pipeline",
				DependsOn:    []string{"fork"},
			},
		},
	}

	_, _, err := workflow.ParseAndImport(context.TODO(), db, cache, proj, nil, input, u, workflow.ImportOptions{Force: true, FromRepository: repofullname})
	assert.NoError(t, err)

	w, errW := workflow.Load(context.TODO(), db, cache, proj, input.Name, u, workflow.LoadOptions{})
	assert.NoError(t, errW)
	assert.NotNil(t, w)

	b, _ := json.Marshal(w)
	t.Logf("Workflow = \n%s", string(b))

	assert.Equal(t, w.FromRepository, repofullname)
	assert.Len(t, w.WorkflowData.Node.Triggers, 2)
	if w.WorkflowData.Node.Triggers[0].ChildNode.Type == "fork" {
		assert.Equal(t, w.WorkflowData.Node.Triggers[0].ChildNode.Name, "fork")
		assert.Len(t, w.WorkflowData.Node.Triggers[0].ChildNode.Triggers, 1)
		assert.Equal(t, w.WorkflowData.Node.Triggers[0].ChildNode.Triggers[0].ChildNode.Name, "third")
	} else {
		assert.Equal(t, w.WorkflowData.Node.Triggers[1].ChildNode.Name, "fork")
		assert.Len(t, w.WorkflowData.Node.Triggers[1].ChildNode.Triggers, 1)
		assert.Equal(t, w.WorkflowData.Node.Triggers[1].ChildNode.Triggers[0].ChildNode.Name, "third")
	}
}
