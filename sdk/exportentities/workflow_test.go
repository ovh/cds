package exportentities

import (
	"testing"

	"github.com/fsamin/go-dump"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func TestWorkflow_checkDependencies(t *testing.T) {
	type fields struct {
		Name            string
		Version         string
		Workflow        map[string]NodeEntry
		Hooks           map[string][]HookEntry
		DependsOn       []string
		Conditions      *sdk.WorkflowNodeConditions
		When            []string
		PipelineName    string
		ApplicationName string
		EnvironmentName string
		PipelineHooks   []HookEntry
		Permissions     map[string]int
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
				Name:            tt.fields.Name,
				Version:         tt.fields.Version,
				Workflow:        tt.fields.Workflow,
				Hooks:           tt.fields.Hooks,
				DependsOn:       tt.fields.DependsOn,
				Conditions:      tt.fields.Conditions,
				When:            tt.fields.When,
				PipelineName:    tt.fields.PipelineName,
				ApplicationName: tt.fields.ApplicationName,
				EnvironmentName: tt.fields.EnvironmentName,
				PipelineHooks:   tt.fields.PipelineHooks,
				Permissions:     tt.fields.Permissions,
			}
			if err := w.checkDependencies(); (err != nil) != tt.wantErr {
				t.Errorf("Workflow.checkDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkflow_checkValidity(t *testing.T) {
	type fields struct {
		Name            string
		Version         string
		Workflow        map[string]NodeEntry
		Hooks           map[string][]HookEntry
		DependsOn       []string
		Conditions      *sdk.WorkflowNodeConditions
		When            []string
		PipelineName    string
		ApplicationName string
		EnvironmentName string
		PipelineHooks   []HookEntry
		Permissions     map[string]int
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
				Name:            tt.fields.Name,
				Version:         tt.fields.Version,
				Workflow:        tt.fields.Workflow,
				Hooks:           tt.fields.Hooks,
				DependsOn:       tt.fields.DependsOn,
				Conditions:      tt.fields.Conditions,
				When:            tt.fields.When,
				PipelineName:    tt.fields.PipelineName,
				ApplicationName: tt.fields.ApplicationName,
				EnvironmentName: tt.fields.EnvironmentName,
				PipelineHooks:   tt.fields.PipelineHooks,
				Permissions:     tt.fields.Permissions,
			}
			if err := w.checkValidity(); (err != nil) != tt.wantErr {
				t.Errorf("Workflow.checkValidity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkflow_GetWorkflow(t *testing.T) {
	type fields struct {
		Name            string
		Version         string
		Workflow        map[string]NodeEntry
		Hooks           map[string][]HookEntry
		DependsOn       []string
		Conditions      *sdk.WorkflowNodeConditions
		When            []string
		PipelineName    string
		ApplicationName string
		EnvironmentName string
		PipelineHooks   []HookEntry
		Permissions     map[string]int
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
				PipelineHooks: []HookEntry{
					{
						Model: "scheduler",
						Config: map[string]string{
							"crontab": "* * * * *",
						},
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				Root: &sdk.WorkflowNode{
					Name: "pipeline",
					Ref:  "pipeline",
					Pipeline: sdk.Pipeline{
						Name: "pipeline",
					},
					Hooks: []sdk.WorkflowNodeHook{
						{
							WorkflowHookModel: sdk.WorkflowHookModel{
								Name: "scheduler",
							},
							Config: sdk.WorkflowNodeHookConfig{
								"crontab": sdk.WorkflowNodeHookConfigValue{
									Value:        "* * * * *",
									Configurable: true,
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
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				Root: &sdk.WorkflowNode{
					Name: "root",
					Ref:  "root",
					Pipeline: sdk.Pipeline{
						Name: "pipeline-root",
					},
					Triggers: []sdk.WorkflowNodeTrigger{
						{
							WorkflowDestNode: sdk.WorkflowNode{
								Name: "child",
								Ref:  "child",
								Pipeline: sdk.Pipeline{
									Name: "pipeline-child",
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
			},
			wantErr: false,
			want: sdk.Workflow{
				Root: &sdk.WorkflowNode{
					Name: "root",
					Ref:  "root",
					Pipeline: sdk.Pipeline{
						Name: "pipeline-root",
					},
					Triggers: []sdk.WorkflowNodeTrigger{
						{
							WorkflowDestNode: sdk.WorkflowNode{
								Name: "child",
								Ref:  "child",
								Pipeline: sdk.Pipeline{
									Name: "pipeline-child",
								},
							},
						},
					},
				},
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
				Root: &sdk.WorkflowNode{
					Name: "root",
					Ref:  "root",
					Pipeline: sdk.Pipeline{
						Name: "pipeline-root",
					},
					Triggers: []sdk.WorkflowNodeTrigger{
						{
							WorkflowDestNode: sdk.WorkflowNode{
								Name: "first",
								Ref:  "first",
								Pipeline: sdk.Pipeline{
									Name: "pipeline-child",
								},
								Triggers: []sdk.WorkflowNodeTrigger{
									{
										WorkflowDestNode: sdk.WorkflowNode{
											Name: "second",
											Ref:  "second",
											Pipeline: sdk.Pipeline{
												Name: "pipeline-child",
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
							Model: "scheduler",
							Config: map[string]string{
								"crontab": "* * * * *",
							},
						},
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
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
								Name: "scheduler",
							},
							Config: sdk.WorkflowNodeHookConfig{
								"crontab": sdk.WorkflowNodeHookConfigValue{
									Value:        "* * * * *",
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
	}
	for _, tt := range tsts {
		t.Run(tt.name, func(t *testing.T) {
			w := Workflow{
				Name:            tt.fields.Name,
				Version:         tt.fields.Version,
				Workflow:        tt.fields.Workflow,
				Hooks:           tt.fields.Hooks,
				DependsOn:       tt.fields.DependsOn,
				Conditions:      tt.fields.Conditions,
				When:            tt.fields.When,
				PipelineName:    tt.fields.PipelineName,
				ApplicationName: tt.fields.ApplicationName,
				EnvironmentName: tt.fields.EnvironmentName,
				PipelineHooks:   tt.fields.PipelineHooks,
				Permissions:     tt.fields.Permissions,
			}
			got, err := w.GetWorkflow()
			if (err != nil) != tt.wantErr {
				t.Errorf("Workflow.GetWorkflow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			expextedValues, _ := dump.ToStringMap(tt.want)
			actualValues, _ := dump.ToStringMap(got)
			for expectedKey, expectedValue := range expextedValues {
				actualValue, ok := actualValues[expectedKey]
				assert.True(t, ok, "%s not found", expectedKey)
				assert.Equal(t, expectedValue, actualValue, "value %s doesn't match. Got %s but want %s", expectedKey, actualValue, expectedValue)
			}

			for actualKey := range actualValues {
				_, ok := expextedValues[actualKey]
				assert.True(t, ok, "got %s, but not found is expected workflow")
			}
		})
	}
}
