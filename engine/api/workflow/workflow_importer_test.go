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
)

func TestImport(t *testing.T) {
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
					Name: "test-1",
					Root: &sdk.WorkflowNode{
						Name: "pipeline",
						Ref:  "pipeline",
						Pipeline: sdk.Pipeline{
							Name: "pipeline",
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
					Root: &sdk.WorkflowNode{
						Name: "pipeline",
						Ref:  "pipeline",
						Pipeline: sdk.Pipeline{
							Name: "pipeline",
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
					Root: &sdk.WorkflowNode{
						Name: "pipeline",
						Ref:  "pipeline",
						Pipeline: sdk.Pipeline{
							Name: "pipeline",
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
					Root: &sdk.WorkflowNode{
						Name: "pipeline",
						Ref:  "pipeline",
						Pipeline: sdk.Pipeline{
							Name: "pipeline",
						},
						Context: &sdk.WorkflowNodeContext{
							Application: &sdk.Application{
								Name: app.Name,
							},
							Environment: &sdk.Environment{
								Name: env.Name,
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
					Root: &sdk.WorkflowNode{
						Name: "pipeline",
						Ref:  "pipeline",
						Pipeline: sdk.Pipeline{
							Name: "pipeline",
						},
						Triggers: []sdk.WorkflowNodeTrigger{
							{
								WorkflowDestNode: sdk.WorkflowNode{
									Name: "child",
									Ref:  "child",
									Pipeline: sdk.Pipeline{
										Name: "pipeline",
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
					Root: &sdk.WorkflowNode{
						Name: "pipeline",
						Ref:  "pipeline",
						Pipeline: sdk.Pipeline{
							Name: "pipeline",
						},
						Triggers: []sdk.WorkflowNodeTrigger{
							{
								WorkflowDestNode: sdk.WorkflowNode{
									Name: "child",
									Ref:  "child",
									Pipeline: sdk.Pipeline{
										Name: "pipeline",
									},
								},
							},
							{
								WorkflowDestNode: sdk.WorkflowNode{
									Name: "second-child",
									Ref:  "second-child",
									Pipeline: sdk.Pipeline{
										Name: "pipeline",
									},
									Context: &sdk.WorkflowNodeContext{
										Application: &sdk.Application{
											Name: app.Name,
										},
										Environment: &sdk.Environment{
											Name: env.Name,
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
					Root: &sdk.WorkflowNode{
						Name: "A",
						Ref:  "A",
						Pipeline: sdk.Pipeline{
							Name: "pipeline",
						},
						Triggers: []sdk.WorkflowNodeTrigger{
							{
								WorkflowDestNode: sdk.WorkflowNode{
									Name: "B",
									Ref:  "B",
									Pipeline: sdk.Pipeline{
										Name: "pipeline",
									},
								},
							},
							{
								WorkflowDestNode: sdk.WorkflowNode{
									Name: "C",
									Ref:  "C",
									Pipeline: sdk.Pipeline{
										Name: "pipeline",
									},
								},
							},
						},
						Hooks: []sdk.WorkflowNodeHook{
							{
								WorkflowHookModel: sdk.WorkflowHookModel{
									Name: "Scheduler",
								},
								Config: sdk.WorkflowNodeHookConfig{
									"cron": sdk.WorkflowNodeHookConfigValue{
										Value:        "* * * * *",
										Configurable: true,
									},
									"timezone": sdk.WorkflowNodeHookConfigValue{
										Value:        "UTC",
										Configurable: true,
									},
								},
							},
						},
					},
					Joins: []sdk.WorkflowNodeJoin{
						{
							SourceNodeRefs: []string{"B", "C"},
							Triggers: []sdk.WorkflowNodeJoinTrigger{
								{
									WorkflowDestNode: sdk.WorkflowNode{
										Name: "D",
										Ref:  "D",
										Pipeline: sdk.Pipeline{
											Name: "pipeline",
										},
									},
								},
								{
									WorkflowDestNode: sdk.WorkflowNode{
										Name: "E",
										Ref:  "E",
										Pipeline: sdk.Pipeline{
											Name: "pipeline",
										},
									},
								},
								{
									WorkflowDestNode: sdk.WorkflowNode{
										Name: "F",
										Ref:  "F",
										Pipeline: sdk.Pipeline{
											Name: "pipeline",
										},
									},
								},
							},
						},
						{
							SourceNodeRefs: []string{"D", "E"},
							Triggers: []sdk.WorkflowNodeJoinTrigger{
								{
									WorkflowDestNode: sdk.WorkflowNode{
										Name: "G",
										Ref:  "G",
										Pipeline: sdk.Pipeline{
											Name: "pipeline",
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
					Root: &sdk.WorkflowNode{
						Name: "pipeline",
						Ref:  "pipeline",
						Pipeline: sdk.Pipeline{
							Name: "pipeline-error",
						},
						Context: &sdk.WorkflowNodeContext{
							Application: &sdk.Application{
								Name: "app-error",
							},
							Environment: &sdk.Environment{
								Name: "env-error",
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
			if err := workflow.Import(db, cache, proj, tt.args.w, u, tt.args.force, nil); err != nil {
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
