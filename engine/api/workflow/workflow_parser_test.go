package workflow_test

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"

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
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, cache, proj, &pip, u))

	//Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
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
					"root": exportentities.NodeEntry{
						PipelineName: "pipeline",
					},
					"first": exportentities.NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"root"},
					},
					"second": exportentities.NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"first"},
					},
					"fork": exportentities.NodeEntry{
						DependsOn: []string{"root"},
					},
					"third": exportentities.NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"fork"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := workflow.ParseAndImport(context.TODO(), db, cache, proj, tt.input, u, workflow.ImportOptions{DryRun: false, Force: true})
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAndImport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				w, errW := workflow.Load(context.TODO(), db, cache, proj, tt.input.Name, u, workflow.LoadOptions{})
				assert.NoError(t, errW)
				b, _ := json.Marshal(w)
				t.Logf("Workflow = \n%s", string(b))

				if tt.name == "test-1" {
					assert.Len(t, w.Root.Forks, 1)
					assert.Equal(t, w.Root.Forks[0].Name, "fork")
					assert.Len(t, w.Root.Forks[0].Triggers, 1)
					assert.Equal(t, w.Root.Forks[0].Triggers[0].WorkflowDestNode.Name, "third")
				}

			}

		})
	}
}
