package v1_test

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/ovh/cds/sdk/exportentities"
	v1 "github.com/ovh/cds/sdk/exportentities/v1"

	"github.com/fsamin/go-dump"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestWorkflow_checkDependencies(t *testing.T) {
	type fields struct {
		Name                   string
		Description            string
		Version                string
		Workflow               map[string]v1.NodeEntry
		Hooks                  map[string][]v1.HookEntry
		Conditions             *v1.ConditionEntry
		When                   []string
		PipelineName           string
		ApplicationName        string
		EnvironmentName        string
		ProjectIntegrationName string
		PipelineHooks          []v1.HookEntry
		Permissions            map[string]int
		HistoryLength          int64
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
			},
			wantErr: false,
		},
		{
			name: "Complex Workflow with a dependency should not raise an error",
			fields: fields{
				Workflow: map[string]v1.NodeEntry{
					"root": {
						PipelineName: "pipeline",
					},
					"child": {
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
				Workflow: map[string]v1.NodeEntry{
					"root": {
						PipelineName: "pipeline",
					},
					"first-child": {
						PipelineName: "pipeline",
						DependsOn:    []string{"root"},
					},
					"second-child": {
						PipelineName: "pipeline",
						DependsOn:    []string{"root"},
					},
					"third-child": {
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
			w := v1.Workflow{
				Name:                   tt.fields.Name,
				Description:            tt.fields.Description,
				Version:                tt.fields.Version,
				Workflow:               tt.fields.Workflow,
				Hooks:                  tt.fields.Hooks,
				Conditions:             tt.fields.Conditions,
				When:                   tt.fields.When,
				PipelineName:           tt.fields.PipelineName,
				ApplicationName:        tt.fields.ApplicationName,
				EnvironmentName:        tt.fields.EnvironmentName,
				ProjectIntegrationName: tt.fields.ProjectIntegrationName,
				PipelineHooks:          tt.fields.PipelineHooks,
				Permissions:            tt.fields.Permissions,
			}
			if err := w.CheckDependencies(); (err != nil) != tt.wantErr {
				t.Errorf("Workflow.checkDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkflow_checkValidity(t *testing.T) {
	type fields struct {
		Name                   string
		Version                string
		Workflow               map[string]v1.NodeEntry
		Hooks                  map[string][]v1.HookEntry
		DependsOn              []string
		Conditions             *v1.ConditionEntry
		When                   []string
		PipelineName           string
		ApplicationName        string
		EnvironmentName        string
		ProjectIntegrationName string
		PipelineHooks          []v1.HookEntry
		Permissions            map[string]int
		OneAtATime             *bool
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
				Workflow: map[string]v1.NodeEntry{
					"root": {
						PipelineName: "pipeline",
					},
					"child": {
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
				Name: "myworkflow",
				Workflow: map[string]v1.NodeEntry{
					"root": {
						PipelineName: "pipeline",
					},
					"child": {
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
				Name:         "myworkflow",
				PipelineName: "pipeline",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := v1.Workflow{
				Name:                   tt.fields.Name,
				Version:                tt.fields.Version,
				Workflow:               tt.fields.Workflow,
				Hooks:                  tt.fields.Hooks,
				Conditions:             tt.fields.Conditions,
				When:                   tt.fields.When,
				PipelineName:           tt.fields.PipelineName,
				ApplicationName:        tt.fields.ApplicationName,
				EnvironmentName:        tt.fields.EnvironmentName,
				ProjectIntegrationName: tt.fields.ProjectIntegrationName,
				PipelineHooks:          tt.fields.PipelineHooks,
				Permissions:            tt.fields.Permissions,
				OneAtATime:             tt.fields.OneAtATime,
			}
			if err := w.CheckValidity(); (err != nil) != tt.wantErr {
				t.Errorf("Workflow.checkValidity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkflow_GetWorkflow(t *testing.T) {
	strue := true
	type fields struct {
		Name                   string
		Description            string
		Version                string
		Workflow               map[string]v1.NodeEntry
		Hooks                  map[string][]v1.HookEntry
		DependsOn              []string
		Conditions             *v1.ConditionEntry
		When                   []string
		PipelineName           string
		ApplicationName        string
		EnvironmentName        string
		ProjectIntegrationName string
		PipelineHooks          []v1.HookEntry
		Permissions            map[string]int
		HistoryLength          int64
		OneAtATime             *bool
	}
	tsts := []struct {
		name    string
		fields  fields
		want    sdk.Workflow
		wantErr bool
	}{
		// pipeline
		{
			name: "Simple workflow with mutex should not raise an error",
			fields: fields{
				Version:      exportentities.WorkflowVersion1,
				Name:         "myworkflow",
				PipelineName: "pipeline",
				OneAtATime:   &strue,
			},
			wantErr: false,
			want: sdk.Workflow{
				Name: "myworkflow",
				WorkflowData: sdk.WorkflowData{
					Node: sdk.Node{
						Name: "pipeline",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineName: "pipeline",
							Mutex:        true,
						},
					},
				},
			},
		},
		// pipeline
		{
			name: "Simple workflow should not raise an error",
			fields: fields{
				Version:      exportentities.ActionVersion1,
				Name:         "myworkflow",
				Description:  "this is my description",
				PipelineName: "pipeline",
				PipelineHooks: []v1.HookEntry{
					{
						Model: "Scheduler",
						Config: map[string]string{
							"crontab": "* * * * *",
							"payload": "{}",
						},
						Conditions: &sdk.WorkflowNodeConditions{
							LuaScript: "return true",
						},
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				Name:        "myworkflow",
				Description: "this is my description",
				WorkflowData: sdk.WorkflowData{
					Node: sdk.Node{
						Name: "pipeline",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineName: "pipeline",
						},
						Hooks: []sdk.NodeHook{
							{
								HookModelName: "Scheduler",
								Conditions: sdk.WorkflowNodeConditions{
									LuaScript: "return true",
								},
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
		// hook conditions
		{
			name: "Workflow with multiple nodes should display hook conditions",
			fields: fields{
				Name:    "myworkflow",
				Version: exportentities.ActionVersion1,
				Workflow: map[string]v1.NodeEntry{
					"root": {
						PipelineName: "pipeline-root",
					},
					"child": {
						PipelineName: "pipeline-child",
						DependsOn:    []string{"root"},
						OneAtATime:   &v1.True,
					},
				},
				Hooks: map[string][]v1.HookEntry{
					"root": []v1.HookEntry{{
						Model: "Scheduler",
						Config: map[string]string{
							"crontab": "* * * * *",
							"payload": "{}",
						},
						Conditions: &sdk.WorkflowNodeConditions{
							LuaScript: "return true",
						},
					}},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				Name: "myworkflow",
				WorkflowData: sdk.WorkflowData{
					Node: sdk.Node{
						Name: "root",
						Type: "pipeline",
						Hooks: []sdk.NodeHook{
							{
								HookModelName: "Scheduler",
								Conditions: sdk.WorkflowNodeConditions{
									LuaScript: "return true",
								},
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
						Context: &sdk.NodeContext{
							PipelineName: "pipeline-root",
						},
						Triggers: []sdk.NodeTrigger{
							{
								ChildNode: sdk.Node{
									Name: "child",
									Ref:  "child",
									Type: "pipeline",
									Context: &sdk.NodeContext{
										PipelineName: "pipeline-child",
										Mutex:        true,
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
			name: "Complexe workflow without joins and mutex should not raise an error",
			fields: fields{
				Name:    "myworkflow",
				Version: exportentities.ActionVersion1,
				Workflow: map[string]v1.NodeEntry{
					"root": {
						PipelineName: "pipeline-root",
					},
					"child": {
						PipelineName: "pipeline-child",
						DependsOn:    []string{"root"},
						OneAtATime:   &v1.True,
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				Name: "myworkflow",
				WorkflowData: sdk.WorkflowData{
					Node: sdk.Node{
						Name: "root",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineName: "pipeline-root",
						},
						Triggers: []sdk.NodeTrigger{
							{
								ChildNode: sdk.Node{
									Name: "child",
									Ref:  "child",
									Type: "pipeline",
									Context: &sdk.NodeContext{
										PipelineName: "pipeline-child",
										Mutex:        true,
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
			name: "Complexe workflow without joins with a default payload on a non root node should raise an error",
			fields: fields{
				Name:    "myworkflow",
				Version: exportentities.ActionVersion1,
				Workflow: map[string]v1.NodeEntry{
					"root": {
						PipelineName: "pipeline-root",
					},
					"child": {
						PipelineName: "pipeline-child",
						DependsOn:    []string{"root"},
						Payload: map[string]interface{}{
							"test": "content",
						},
						OneAtATime: &v1.True,
					},
				},
			},
			wantErr: true,
		},
		// root(pipeline-root) -> child(pipeline-child)
		{
			name: "Complexe workflow unordered without joins should not raise an error",
			fields: fields{
				Name:    "myworkflow",
				Version: exportentities.ActionVersion1,
				Workflow: map[string]v1.NodeEntry{
					"child": {
						PipelineName: "pipeline-child",
						DependsOn:    []string{"root"},
					},
					"root": {
						PipelineName: "pipeline-root",
					},
				},
				HistoryLength: 25,
			},
			wantErr: false,
			want: sdk.Workflow{
				Name:          "myworkflow",
				HistoryLength: 25,
				WorkflowData: sdk.WorkflowData{
					Node: sdk.Node{
						Name: "root",
						Ref:  "root",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineName: "pipeline-root",
						},
						Triggers: []sdk.NodeTrigger{
							{
								ChildNode: sdk.Node{
									Name: "child",
									Ref:  "child",
									Type: "pipeline",
									Context: &sdk.NodeContext{
										PipelineName: "pipeline-child",
									},
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
				Name:    "myworkflow",
				Version: exportentities.ActionVersion1,
				Workflow: map[string]v1.NodeEntry{
					"root": {
						PipelineName: "pipeline-root",
					},
					"first": {
						PipelineName: "pipeline-child",
						DependsOn:    []string{"root"},
					},
					"second": {
						PipelineName: "pipeline-child",
						DependsOn:    []string{"first"},
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				Name: "myworkflow",
				WorkflowData: sdk.WorkflowData{
					Node: sdk.Node{
						Name: "root",
						Ref:  "root",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineName: "pipeline-root",
						},
						Triggers: []sdk.NodeTrigger{
							{
								ChildNode: sdk.Node{
									Name: "first",
									Ref:  "first",
									Type: "pipeline",
									Context: &sdk.NodeContext{
										PipelineName: "pipeline-child",
									},

									Triggers: []sdk.NodeTrigger{
										{
											ChildNode: sdk.Node{
												Name: "second",
												Ref:  "second",
												Type: "pipeline",
												Context: &sdk.NodeContext{
													PipelineName: "pipeline-child",
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
				Name:    "myworkflow",
				Version: exportentities.ActionVersion1,
				Workflow: map[string]v1.NodeEntry{
					"A": {
						PipelineName: "pipeline",
					},
					"B": {
						PipelineName: "pipeline",
						DependsOn:    []string{"A"},
					},
					"C": {
						PipelineName: "pipeline",
						DependsOn:    []string{"A"},
					},
					"D": {
						PipelineName: "pipeline",
						DependsOn:    []string{"B", "C"},
					},
					"E": {
						PipelineName: "pipeline",
						DependsOn:    []string{"B", "C"},
					},
					"F": {
						PipelineName: "pipeline",
						DependsOn:    []string{"B", "C"},
					},
					"G": {
						PipelineName: "pipeline",
						DependsOn:    []string{"D", "E"},
					},
				},
				Hooks: map[string][]v1.HookEntry{
					"A": {
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
				Name: "myworkflow",
				WorkflowData: sdk.WorkflowData{
					Node: sdk.Node{
						Name: "A",
						Ref:  "A",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineName: "pipeline",
						},
						Triggers: []sdk.NodeTrigger{
							{
								ChildNode: sdk.Node{
									Name: "B",
									Ref:  "B",
									Type: "pipeline",
									Context: &sdk.NodeContext{
										PipelineName: "pipeline",
									},
								},
							},
							{
								ChildNode: sdk.Node{
									Name: "C",
									Ref:  "C",
									Type: "pipeline",
									Context: &sdk.NodeContext{
										PipelineName: "pipeline",
									},
								},
							},
						},
						Hooks: []sdk.NodeHook{
							{
								HookModelName: "Scheduler",
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
											PipelineName: "pipeline",
										},
									},
								},
								{
									ChildNode: sdk.Node{
										Name: "E",
										Ref:  "E",
										Type: "pipeline",
										Context: &sdk.NodeContext{
											PipelineName: "pipeline",
										},
									},
								},
								{
									ChildNode: sdk.Node{
										Name: "F",
										Ref:  "F",
										Type: "pipeline",
										Context: &sdk.NodeContext{
											PipelineName: "pipeline",
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
											PipelineName: "pipeline",
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
			name: "Complex workflow with integration should not raise an error",
			fields: fields{
				Name:                   "myworkflow",
				Version:                exportentities.ActionVersion1,
				PipelineName:           "pipeline",
				ProjectIntegrationName: "integration",
			},
			wantErr: false,
			want: sdk.Workflow{
				Name: "myworkflow",
				WorkflowData: sdk.WorkflowData{
					Node: sdk.Node{
						Name: "pipeline",
						Ref:  "pipeline",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineName:           "pipeline",
							ProjectIntegrationName: "integration",
						},
					},
				},
			},
		},
		{
			name: "Root and a outgoing hook should not raise an error",
			fields: fields{
				Name:    "myworkflow",
				Version: exportentities.ActionVersion1,
				Workflow: map[string]v1.NodeEntry{
					"A": {
						PipelineName: "pipeline",
					},
					"B": {
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
				Name: "myworkflow",
				WorkflowData: sdk.WorkflowData{
					Node: sdk.Node{
						Name: "A",
						Ref:  "pipeline",
						Type: "pipeline",
						Context: &sdk.NodeContext{
							PipelineName: "pipeline",
						},
						Triggers: []sdk.NodeTrigger{
							{
								ChildNode: sdk.Node{
									Name:    "B",
									Type:    sdk.NodeTypeOutGoingHook,
									Context: &sdk.NodeContext{},
									OutGoingHookContext: &sdk.NodeOutGoingHook{
										HookModelName: "webhook",
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
		{
			name: "Workflow v2 with no nodes",
			fields: fields{
				Name:     "myworkflow",
				Version:  exportentities.WorkflowVersion2,
				Workflow: map[string]v1.NodeEntry{},
			},
			wantErr: true,
		},
		{
			name: "Workflow v1 with no nodes",
			fields: fields{
				Name:     "myworkflow",
				Version:  exportentities.WorkflowVersion1,
				Workflow: map[string]v1.NodeEntry{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tsts {
		t.Run(tt.name, func(t *testing.T) {
			w := v1.Workflow{
				Name:                   tt.fields.Name,
				Description:            tt.fields.Description,
				Version:                tt.fields.Version,
				Workflow:               tt.fields.Workflow,
				Hooks:                  tt.fields.Hooks,
				Conditions:             tt.fields.Conditions,
				When:                   tt.fields.When,
				PipelineName:           tt.fields.PipelineName,
				ApplicationName:        tt.fields.ApplicationName,
				EnvironmentName:        tt.fields.EnvironmentName,
				ProjectIntegrationName: tt.fields.ProjectIntegrationName,
				PipelineHooks:          tt.fields.PipelineHooks,
				Permissions:            tt.fields.Permissions,
				HistoryLength:          &tt.fields.HistoryLength,
				OneAtATime:             tt.fields.OneAtATime,
			}

			got, err := exportentities.ParseWorkflow(context.TODO(), w)
			if (err != nil) != tt.wantErr {
				t.Errorf("Workflow.GetWorkflow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			got.HookModels = nil
			got.OutGoingHookModels = nil
			got.Applications = nil
			got.Pipelines = nil
			got.Environments = nil
			got.ProjectIntegrations = nil

			expectedValues, _ := dump.ToStringMap(tt.want)
			actualValues, _ := dump.ToStringMap(got)

			var keysExpectedValues []string
			for k := range expectedValues {
				keysExpectedValues = append(keysExpectedValues, k)
			}
			sort.Strings(keysExpectedValues)

			for _, expectedKey := range keysExpectedValues {
				expectedValue := expectedValues[expectedKey]
				actualValue, ok := actualValues[expectedKey]
				if strings.Contains(expectedKey, ".Ref") {
					assert.NotEmpty(t, actualValue, "value %s is empty but should not be empty", expectedKey)
				} else {
					assert.True(t, ok, "%s not found", expectedKey)
					assert.Equal(t, expectedValue, actualValue, "value %s doesn't match. Got %s but want %s", expectedKey, actualValue, expectedValue)
				}
			}

			for actualKey := range actualValues {
				_, ok := expectedValues[actualKey]
				assert.True(t, ok, "got %s, but not found is expected workflow", actualKey)
			}
		})
	}
}
