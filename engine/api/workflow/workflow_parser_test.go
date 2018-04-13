package workflow_test

import (
	"encoding/json"
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
	db, cache := test.SetupPG(t)
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
			name: "Insert workflow with 2 children",
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
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := workflow.ParseAndImport(db, cache, proj, tt.input, true, u, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAndImport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				w, _ := workflow.Load(db, cache, proj.Key, tt.input.Name, u, workflow.LoadOptions{})
				if w != nil {
					b, _ := json.Marshal(w)
					t.Logf("Workflow = \n%s", string(b))
				}
			}

		})
	}
}
