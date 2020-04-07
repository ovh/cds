package workflow_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

type mockHTTPClient struct {
	f func(r *http.Request) (*http.Response, error) // nolint
}

func (h *mockHTTPClient) Do(*http.Request) (*http.Response, error) {
	body := ioutil.NopCloser(bytes.NewReader([]byte("{}")))
	return &http.Response{Body: body}, nil
}

func TestImport(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()
	u, _ := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	srvs, _ := services.LoadAll(context.TODO(), db)
	for _, srv := range srvs {
		if err := services.Delete(db, &srv); err != nil {
			log.Fatalf("unable to delete service %s", srv.Name)
		}
	}

	assets.InsertService(t, db, "service_test"+sdk.RandomString(5), services.TypeHooks)

	//Mock HTTPClient from services package
	services.HTTPClient = &mockHTTPClient{}

	//Pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pipeline",
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip))

	//Pipeline
	pipparam := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pipeline-with-param",
	}
	sdk.AddParameter(&pipparam.Parameter, "name", sdk.StringParameter, "value")

	test.NoError(t, pipeline.InsertPipeline(db, &pipparam))
	//Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	test.NoError(t, application.Insert(db, cache, *proj, app))

	//Environment
	envName := sdk.RandomString(10)
	env := &sdk.Environment{
		ProjectID: proj.ID,
		Name:      envName,
	}
	test.NoError(t, environment.InsertEnvironment(db, env))

	//Reload project
	proj, _ = project.Load(db, cache, proj.Key, project.LoadOptions.WithApplications, project.LoadOptions.WithEnvironments, project.LoadOptions.WithPipelines)

	test.NoError(t, workflow.CreateBuiltinWorkflowHookModels(db))
	hookModels, err := workflow.LoadHookModels(db)
	test.NoError(t, err)

	var schedulModelID int64
	for _, m := range hookModels {
		if m.Name == sdk.SchedulerModel.Name {
			schedulModelID = m.ID
		}
	}

	type args struct {
		w     *sdk.Workflow
		force bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "simple workflow insertion",
			args: args{
				w: &sdk.Workflow{
					Name:      "test-1",
					Metadata:  sdk.Metadata{"triggered_by": "bla"},
					PurgeTags: []string{"aa", "bb"},
					WorkflowData: sdk.WorkflowData{
						Node: sdk.Node{
							Name: "pipeline",
							Ref:  "pipeline",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
				force: false,
			},
			wantErr: false,
		},
		{
			name: "same workflow insertion should failed with 409",
			args: args{
				w: &sdk.Workflow{
					Name: "test-1",
					WorkflowData: sdk.WorkflowData{
						Node: sdk.Node{
							Name: "pipeline",
							Ref:  "pipeline",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
				force: false,
			},
			wantErr: true,
		},
		{
			name: "workflow update should succeed with force",
			args: args{
				w: &sdk.Workflow{
					Name: "test-1",
					WorkflowData: sdk.WorkflowData{
						Node: sdk.Node{
							Name: "pipeline",
							Ref:  "pipeline",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
						},
					},
				},
				force: true,
			},
			wantErr: false,
		},
		{
			name: "workflow insertion with app and env",
			args: args{
				w: &sdk.Workflow{
					Name: "test-2",
					WorkflowData: sdk.WorkflowData{
						Node: sdk.Node{
							Name: "pipeline",
							Ref:  "pipeline",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID:    pip.ID,
								ApplicationID: app.ID,
								EnvironmentID: env.ID,
							},
						},
					},
				},
				force: false,
			},
			wantErr: false,
		},
		{
			name: "workflow insertion with a trigger",
			args: args{
				w: &sdk.Workflow{
					Name: "test-3",
					WorkflowData: sdk.WorkflowData{
						Node: sdk.Node{
							Name: "pipeline",
							Ref:  "pipeline",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
							Triggers: []sdk.NodeTrigger{
								{
									ChildNode: sdk.Node{
										Name: "child",
										Ref:  "child",
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID: pip.ID,
										},
									},
								},
							},
						},
					},
				},
				force: false,
			},
			wantErr: false,
		},
		{
			name: "workflow update with a trigger",
			args: args{
				w: &sdk.Workflow{
					Name: "test-3",
					WorkflowData: sdk.WorkflowData{
						Node: sdk.Node{
							Name: "pipeline",
							Ref:  "pipeline",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
							Triggers: []sdk.NodeTrigger{
								{
									ChildNode: sdk.Node{
										Name: "child",
										Ref:  "child",
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID: pip.ID,
										},
									},
								},
								{
									ChildNode: sdk.Node{
										Name: "second-child",
										Ref:  "second-child",
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID:    pip.ID,
											ApplicationID: app.ID,
											EnvironmentID: env.ID,
										},
									},
								},
							},
						},
					},
				},
				force: true,
			},
			wantErr: false,
		}, {
			name: "complexe workflow insert with hook",
			args: args{
				w: &sdk.Workflow{
					Name: "test-4",
					WorkflowData: sdk.WorkflowData{
						Node: sdk.Node{
							Name: "A",
							Ref:  "A",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
							},
							Triggers: []sdk.NodeTrigger{
								{
									ChildNode: sdk.Node{
										Name: "B",
										Ref:  "B",
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID: pip.ID,
										},
									},
								},
								{
									ChildNode: sdk.Node{
										Name: "C",
										Ref:  "C",
										Type: sdk.NodeTypePipeline,
										Context: &sdk.NodeContext{
											PipelineID: pip.ID,
										},
									},
								},
							},
							Hooks: []sdk.NodeHook{
								{
									HookModelID: schedulModelID,
									Config: sdk.WorkflowNodeHookConfig{
										sdk.SchedulerModelCron: sdk.WorkflowNodeHookConfigValue{
											Value:        "* * * * *",
											Configurable: true,
										},
										sdk.SchedulerModelTimezone: sdk.WorkflowNodeHookConfigValue{
											Value:        "UTC",
											Configurable: true,
										},
										sdk.Payload: sdk.WorkflowNodeHookConfigValue{
											Value:        "{}",
											Configurable: true,
										},
									},
								},
							},
						},
						Joins: []sdk.Node{
							{
								Name: "join1",
								Ref:  "join1",
								Type: sdk.NodeTypeJoin,
								JoinContext: []sdk.NodeJoin{
									{
										ParentName: "B",
									},
									{
										ParentName: "C",
									},
								},
								Triggers: []sdk.NodeTrigger{
									{
										ChildNode: sdk.Node{
											Name: "D",
											Ref:  "D",
											Type: sdk.NodeTypePipeline,
											Context: &sdk.NodeContext{
												PipelineID: pip.ID,
											},
										},
									},
									{
										ChildNode: sdk.Node{
											Name: "E",
											Ref:  "E",
											Type: sdk.NodeTypePipeline,
											Context: &sdk.NodeContext{
												PipelineID: pip.ID,
											},
										},
									},
									{
										ChildNode: sdk.Node{
											Name: "F",
											Ref:  "F",
											Type: sdk.NodeTypePipeline,
											Context: &sdk.NodeContext{
												PipelineID: pip.ID,
											},
										},
									},
								},
							},
							{
								Name: "join2",
								Ref:  "join2",
								Type: sdk.NodeTypeJoin,
								JoinContext: []sdk.NodeJoin{
									{
										ParentName: "D",
									},
									{
										ParentName: "E",
									},
								},
								Triggers: []sdk.NodeTrigger{
									{
										ChildNode: sdk.Node{
											Name: "G",
											Ref:  "G",
											Type: sdk.NodeTypePipeline,
											Context: &sdk.NodeContext{
												PipelineID: pip.ID,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "workflow insertion with wrong pip, app and env",
			args: args{
				w: &sdk.Workflow{
					Name: "test-5",
					WorkflowData: sdk.WorkflowData{
						Node: sdk.Node{
							Name: "pipeline",
							Ref:  "pipeline",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID:    99,
								ApplicationID: 99,
								EnvironmentID: 99,
							},
						},
					},
				},
				force: false,
			},
			wantErr: true,
		},
		{
			name: "workflow insertion with pipeline parameter",
			args: args{
				w: &sdk.Workflow{
					Name: "test-6",
					WorkflowData: sdk.WorkflowData{
						Node: sdk.Node{
							Name: "pipeline",
							Ref:  "pipeline",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pipparam.ID,
								DefaultPipelineParameters: []sdk.Parameter{
									{
										Name:  "name",
										Value: "value",
									},
								},
							},
						},
					},
				},
				force: false,
			},
			wantErr: true,
		},
		{
			name: "simple workflow insertion with wrong parameter",
			args: args{
				w: &sdk.Workflow{
					Name: "test-1",
					WorkflowData: sdk.WorkflowData{
						Node: sdk.Node{
							Name: "pipeline",
							Ref:  "pipeline",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID: pip.ID,
								DefaultPipelineParameters: []sdk.Parameter{
									{
										Name:  "name",
										Value: "value",
									},
								},
							},
						},
					},
				},
				force: false,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			workflowExists, err := workflow.Exists(db, proj.Key, tt.args.w.Name)
			if err != nil {
				t.Errorf("%s", err)
			}
			var wf *sdk.Workflow
			if workflowExists {
				wf, err = workflow.Load(context.TODO(), db, cache, *proj, tt.args.w.Name, workflow.LoadOptions{WithIcon: true})
				if err != nil {
					t.Errorf("%s", err)
				}
			}

			if err := workflow.Import(context.TODO(), db, cache, *proj, wf, tt.args.w, u, tt.args.force, nil); err != nil {
				if !tt.wantErr {
					t.Errorf("Import() error = %v, wantErr %v", err, tt.wantErr)
				} else {
					t.Logf("Import() returns error = %v", err)
				}
			} else {
				b, _ := json.Marshal(tt.args.w)
				t.Logf("Success: workflow = \n%s", string(b))
			}
		})
	}
}
