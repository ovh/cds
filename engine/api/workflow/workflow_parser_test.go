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
	proj := assets.InsertTestProject(t, db, cache, key, key)

	//Pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pipeline",
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip))

	//Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	test.NoError(t, application.Insert(db, cache, proj, app))

	//Environment
	envName := sdk.RandomString(10)
	env := &sdk.Environment{
		ProjectID: proj.ID,
		Name:      envName,
	}
	test.NoError(t, environment.InsertEnvironment(db, env))

	//Reload project
	proj, _ = project.Load(db, cache, proj.Key, project.LoadOptions.WithApplications, project.LoadOptions.WithEnvironments, project.LoadOptions.WithPipelines)

	tests := []struct {
		name    string
		input   *exportentities.Workflow
		wantErr bool
	}{
		{
			name: "Insert workflow with 2 children + 1 fork with 1 child",
			input: &exportentities.Workflow{
				Name: "test-1",
				Workflow: map[string]exportentities.NodeEntry{
					"root": {
						PipelineName: "pipeline",
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := workflow.ParseAndImport(context.TODO(), db, cache, proj, nil, tt.input, u, workflow.ImportOptions{Force: true})
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAndImport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				w, errW := workflow.Load(context.TODO(), db, cache, proj, tt.input.Name, workflow.LoadOptions{})
				assert.NoError(t, errW)
				b, _ := json.Marshal(w)
				t.Logf("Workflow = \n%s", string(b))

				if tt.name == "test-1" {
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

			}

		})
	}
}
