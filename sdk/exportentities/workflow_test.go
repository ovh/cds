package exportentities

import (
	"sort"
	"strings"
	"testing"

	"github.com/fsamin/go-dump"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

var True = true

func TestWorkflow_checkDependencies(t *testing.T) {
	type fields struct {
		Name                string
		Description         string
		Version             string
		Workflow            map[string]NodeEntry
		Hooks               map[string][]HookEntry
		DependsOn           []string
		Conditions          *sdk.WorkflowNodeConditions
		When                []string
		PipelineName        string
		ApplicationName     string
		EnvironmentName     string
		ProjectPlatformName string
		PipelineHooks       []HookEntry
		Permissions         map[string]int
		HistoryLength       int64
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Simple Workflow without dependencies should not raise an error",
			fields: fields{
				PipelineName: "pipeline",
				DependsOn:    []string{"non existing"},
			},
			wantErr: true,
		},
		{
			name: "Simple Workflow with an invalid dependency should raise an error",
			fields: fields{
				PipelineName: "pipeline",
				Description:  "here is my description",
			},
			wantErr: false,
		},
		{
			name: "Complex Workflow with a dependency should not raise an error",
			fields: fields{
				Workflow: map[string]NodeEntry{
					"root": NodeEntry{
						PipelineName: "pipeline",
					},
					"child": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"root"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Complex Workflow with a dependencies and a join should not raise an error",
			fields: fields{
				Workflow: map[string]NodeEntry{
					"root": NodeEntry{
						PipelineName: "pipeline",
					},
					"first-child": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"root"},
					},
					"second-child": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"root"},
					},
					"third-child": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"first-child", "second-child"},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := Workflow{
				Name:                tt.fields.Name,
				Description:         tt.fields.Description,
				Version:             tt.fields.Version,
				Workflow:            tt.fields.Workflow,
				Hooks:               tt.fields.Hooks,
				DependsOn:           tt.fields.DependsOn,
				Conditions:          tt.fields.Conditions,
				When:                tt.fields.When,
				PipelineName:        tt.fields.PipelineName,
				ApplicationName:     tt.fields.ApplicationName,
				EnvironmentName:     tt.fields.EnvironmentName,
				ProjectPlatformName: tt.fields.ProjectPlatformName,
				PipelineHooks:       tt.fields.PipelineHooks,
				Permissions:         tt.fields.Permissions,
			}
			if err := w.checkDependencies(); (err != nil) != tt.wantErr {
				t.Errorf("Workflow.checkDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkflow_checkValidity(t *testing.T) {
	type fields struct {
		Name                string
		Version             string
		Workflow            map[string]NodeEntry
		Hooks               map[string][]HookEntry
		DependsOn           []string
		Conditions          *sdk.WorkflowNodeConditions
		When                []string
		PipelineName        string
		ApplicationName     string
		EnvironmentName     string
		ProjectPlatformName string
		PipelineHooks       []HookEntry
		Permissions         map[string]int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should raise an error",
			fields: fields{
				PipelineName: "pipeline",
				Workflow: map[string]NodeEntry{
					"root": NodeEntry{
						PipelineName: "pipeline",
					},
					"child": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"root"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should not raise an error",
			fields: fields{
				Workflow: map[string]NodeEntry{
					"root": NodeEntry{
						PipelineName: "pipeline",
					},
					"child": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"root"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Too simple to raise an error",
			fields: fields{
				PipelineName: "pipeline",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := Workflow{
				Name:                tt.fields.Name,
				Version:             tt.fields.Version,
				Workflow:            tt.fields.Workflow,
				Hooks:               tt.fields.Hooks,
				DependsOn:           tt.fields.DependsOn,
				Conditions:          tt.fields.Conditions,
				When:                tt.fields.When,
				PipelineName:        tt.fields.PipelineName,
				ApplicationName:     tt.fields.ApplicationName,
				EnvironmentName:     tt.fields.EnvironmentName,
				ProjectPlatformName: tt.fields.ProjectPlatformName,
				PipelineHooks:       tt.fields.PipelineHooks,
				Permissions:         tt.fields.Permissions,
			}
			if err := w.checkValidity(); (err != nil) != tt.wantErr {
				t.Errorf("Workflow.checkValidity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkflow_GetWorkflow(t *testing.T) {
	proj := sdk.Project{
		Pipelines: []sdk.Pipeline{
			{
				ID:   1,
				Name: "pipeline",
			},
			{
				ID:   2,
				Name: "pipeline-root",
			},
			{
				ID:   3,
				Name: "pipeline-child",
			},
		},
		Platforms: []sdk.ProjectPlatform{
			{
				ID:   1,
				Name: "platform",
			},
		},
	}
	hooksModels := []sdk.WorkflowHookModel{
		{
			ID:            1,
			Name:          "Scheduler",
			Type:          sdk.WorkflowHookModelBuiltin,
			Identifier:    sdk.SchedulerModel.Identifier,
			Author:        "CDS",
			Icon:          "fa-clock-o",
			DefaultConfig: sdk.SchedulerModel.DefaultConfig,
		},
	}

	outgoingModels := []sdk.WorkflowHookModel{
		{
			ID:   1,
			Name: "webhook",
		},
	}

	type fields struct {
		Name                string
		Description         string
		Version             string
		Workflow            map[string]NodeEntry
		Hooks               map[string][]HookEntry
		DependsOn           []string
		Conditions          *sdk.WorkflowNodeConditions
		When                []string
		PipelineName        string
		ApplicationName     string
		EnvironmentName     string
		ProjectPlatformName string
		PipelineHooks       []HookEntry
		Permissions         map[string]int
		HistoryLength       int64
	}
	tsts := []struct {
		name    string
		fields  fields
		want    sdk.Workflow
		wantErr bool
	}{
		// pipeline
		{
			name: "Simple workflow should not raise an error",
			fields: fields{
				PipelineName: "pipeline",
				Description:  "this is my description",
				PipelineHooks: []HookEntry{
					{
						Model: "Scheduler",
						Config: map[string]string{
							"crontab": "* * * * *",
							"payload": "{}",
						},
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				Description: "this is my description",
				WorkflowData: &sdk.WorkflowData{
					Node: sdk.Node{
						Name: "pipeline",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineID: 1,
						},
						Hooks: []sdk.NodeHook{
							{
								HookModelID: 1,
								Config: sdk.WorkflowNodeHookConfig{
									"crontab": sdk.WorkflowNodeHookConfigValue{
										Value:        "* * * * *",
										Configurable: true,
										Type:         sdk.HookConfigTypeString,
									},
									"payload": sdk.WorkflowNodeHookConfigValue{
										Value:        "{}",
										Configurable: true,
										Type:         sdk.HookConfigTypeString,
									},
								},
							},
						},
					},
				},
			},
		},
		// root(pipeline-root) -> child(pipeline-child)
		{
			name: "Complexe workflow without joins should not raise an error",
			fields: fields{
				Workflow: map[string]NodeEntry{
					"root": NodeEntry{
						PipelineName: "pipeline-root",
					},
					"child": NodeEntry{
						PipelineName: "pipeline-child",
						DependsOn:    []string{"root"},
						OneAtATime:   &True,
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				WorkflowData: &sdk.WorkflowData{
					Node: sdk.Node{
						Name: "root",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineID: 2,
						},
						Triggers: []sdk.NodeTrigger{
							{
								ChildNode: sdk.Node{
									Name: "child",
									Ref:  "child",
									Type: "pipeline",
									Context: &sdk.NodeContext{
										PipelineID: 3,
										Mutex:      true,
									},
								},
							},
						},
					},
				},
			},
		},
		// root(pipeline-root) -> child(pipeline-child)
		{
			name: "Complexe workflow unordered without joins should not raise an error",
			fields: fields{
				Workflow: map[string]NodeEntry{
					"child": NodeEntry{
						PipelineName: "pipeline-child",
						DependsOn:    []string{"root"},
					},
					"root": NodeEntry{
						PipelineName: "pipeline-root",
					},
				},
				HistoryLength: 25,
			},
			wantErr: false,
			want: sdk.Workflow{
				WorkflowData: &sdk.WorkflowData{
					Node: sdk.Node{
						Name: "root",
						Ref:  "root",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineID: 2,
						},
						Triggers: []sdk.NodeTrigger{
							{
								ChildNode: sdk.Node{
									Name: "child",
									Ref:  "child",
									Type: "pipeline",
									Context: &sdk.NodeContext{
										PipelineID: 3,
									},
								},
							},
						},
					},
				},
				HistoryLength: 25,
			},
		},
		// root(pipeline-root) -> first(pipeline-child) -> second(pipeline-child)
		{
			name: "Complexe workflow without joins should not raise an error",
			fields: fields{
				Workflow: map[string]NodeEntry{
					"root": NodeEntry{
						PipelineName: "pipeline-root",
					},
					"first": NodeEntry{
						PipelineName: "pipeline-child",
						DependsOn:    []string{"root"},
					},
					"second": NodeEntry{
						PipelineName: "pipeline-child",
						DependsOn:    []string{"first"},
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				WorkflowData: &sdk.WorkflowData{
					Node: sdk.Node{
						Name: "root",
						Ref:  "root",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineID: 2,
						},
						Triggers: []sdk.NodeTrigger{
							{
								ChildNode: sdk.Node{
									Name: "first",
									Ref:  "first",
									Type: "pipeline",
									Context: &sdk.NodeContext{
										PipelineID: 3,
									},

									Triggers: []sdk.NodeTrigger{
										{
											ChildNode: sdk.Node{
												Name: "second",
												Ref:  "second",
												Type: "pipeline",
												Context: &sdk.NodeContext{
													PipelineID: 3,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		// A(pipeline)(*) -> B(pipeline) -> join -> D(pipeline) -> join -> G(pipeline)
		//                -> C(pipeline) /       -> E(pipeline) /
		//                                       -> F(pipeline)
		{
			name: "Complexe workflow with joins should not raise an error",
			fields: fields{
				Workflow: map[string]NodeEntry{
					"A": NodeEntry{
						PipelineName: "pipeline",
					},
					"B": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"A"},
					},
					"C": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"A"},
					},
					"D": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"B", "C"},
					},
					"E": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"B", "C"},
					},
					"F": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"B", "C"},
					},
					"G": NodeEntry{
						PipelineName: "pipeline",
						DependsOn:    []string{"D", "E"},
					},
				},
				Hooks: map[string][]HookEntry{
					"A": []HookEntry{
						{
							Model: "Scheduler",
							Config: map[string]string{
								"crontab": "* * * * *",
								"payload": "{}",
							},
						},
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				WorkflowData: &sdk.WorkflowData{
					Node: sdk.Node{
						Name: "A",
						Ref:  "A",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineID: 1,
						},
						Triggers: []sdk.NodeTrigger{
							{
								ChildNode: sdk.Node{
									Name: "B",
									Ref:  "B",
									Type: "pipeline",
									Context: &sdk.NodeContext{
										PipelineID: 1,
									},
								},
							},
							{
								ChildNode: sdk.Node{
									Name: "C",
									Ref:  "C",
									Type: "pipeline",
									Context: &sdk.NodeContext{
										PipelineID: 1,
									},
								},
							},
						},
						Hooks: []sdk.NodeHook{
							{
								HookModelID: 1,
								Config: sdk.WorkflowNodeHookConfig{
									"crontab": sdk.WorkflowNodeHookConfigValue{
										Value:        "* * * * *",
										Configurable: true,
										Type:         sdk.HookConfigTypeString,
									},
									"payload": sdk.WorkflowNodeHookConfigValue{
										Value:        "{}",
										Configurable: true,
										Type:         sdk.HookConfigTypeString,
									},
								},
							},
						},
					},
					Joins: []sdk.Node{
						{
							Type: "join",
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
										Type: "pipeline",
										Context: &sdk.NodeContext{
											PipelineID: 1,
										},
									},
								},
								{
									ChildNode: sdk.Node{
										Name: "E",
										Ref:  "E",
										Type: "pipeline",
										Context: &sdk.NodeContext{
											PipelineID: 1,
										},
									},
								},
								{
									ChildNode: sdk.Node{
										Name: "F",
										Ref:  "F",
										Type: "pipeline",
										Context: &sdk.NodeContext{
											PipelineID: 1,
										},
									},
								},
							},
						},
						{
							Type: "join",
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
										Type: "pipeline",
										Context: &sdk.NodeContext{
											PipelineID: 1,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Complex workflow with platform should not raise an error",
			fields: fields{
				PipelineName:        "pipeline",
				ProjectPlatformName: "platform",
			},
			wantErr: false,
			want: sdk.Workflow{
				WorkflowData: &sdk.WorkflowData{
					Node: sdk.Node{
						Name: "pipeline",
						Ref:  "pipeline",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineID:        1,
							ProjectPlatformID: 1,
						},
					},
				},
			},
		},
		{
			name: "Root and a outgoing hook should not raise an error",
			fields: fields{
				Workflow: map[string]NodeEntry{
					"A": NodeEntry{
						PipelineName: "pipeline",
					},
					"B": NodeEntry{
						OutgoingHookModelName: "webhook",
						OutgoingHookConfig: map[string]string{
							"url": "https://www.ovh.com",
						},
						DependsOn: []string{"A"},
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				WorkflowData: &sdk.WorkflowData{
					Node: sdk.Node{
						Name: "A",
						Ref:  "pipeline",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineID: 1,
						},
						Triggers: []sdk.NodeTrigger{
							{
								ChildNode: sdk.Node{
									Name:    "B",
									Type:    sdk.NodeTypeOutGoingHook,
									Context: &sdk.NodeContext{},
									OutGoingHookContext: &sdk.NodeOutGoingHook{
										HookModelID: 1,
										Config: sdk.WorkflowNodeHookConfig{
											"url": sdk.WorkflowNodeHookConfigValue{
												Value: "https://www.ovh.com",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tsts {
		t.Run(tt.name, func(t *testing.T) {
			w := Workflow{
				Name:                tt.fields.Name,
				Description:         tt.fields.Description,
				Version:             tt.fields.Version,
				Workflow:            tt.fields.Workflow,
				Hooks:               tt.fields.Hooks,
				DependsOn:           tt.fields.DependsOn,
				Conditions:          tt.fields.Conditions,
				When:                tt.fields.When,
				PipelineName:        tt.fields.PipelineName,
				ApplicationName:     tt.fields.ApplicationName,
				EnvironmentName:     tt.fields.EnvironmentName,
				ProjectPlatformName: tt.fields.ProjectPlatformName,
				PipelineHooks:       tt.fields.PipelineHooks,
				Permissions:         tt.fields.Permissions,
				HistoryLength:       tt.fields.HistoryLength,
			}
			got, err := w.GetWorkflow(&proj, hooksModels, outgoingModels)
			if (err != nil) != tt.wantErr {
				t.Errorf("Workflow.GetWorkflow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got.HookModels = nil
			got.OutGoingHookModels = nil
			got.Applications = nil
			got.Pipelines = nil
			got.Environments = nil
			got.ProjectPlatforms = nil

			expextedValues, _ := dump.ToStringMap(tt.want)
			actualValues, _ := dump.ToStringMap(got)

			var keysExpextedValues []string
			for k := range expextedValues {
				keysExpextedValues = append(keysExpextedValues, k)
			}
			sort.Strings(keysExpextedValues)

			for _, expectedKey := range keysExpextedValues {
				expectedValue := expextedValues[expectedKey]
				actualValue, ok := actualValues[expectedKey]
				if strings.Contains(expectedKey, ".Ref") {
					assert.NotEmpty(t, actualValue, "value %s is empty but shoud not be empty", expectedKey)
				} else {
					assert.True(t, ok, "%s not found", expectedKey)
					assert.Equal(t, expectedValue, actualValue, "value %s doesn't match. Got %s but want %s", expectedKey, actualValue, expectedValue)
				}
			}

			for actualKey := range actualValues {
				_, ok := expextedValues[actualKey]
				assert.True(t, ok, "got %s, but not found is expected workflow", actualKey)
			}
		})
	}
}
