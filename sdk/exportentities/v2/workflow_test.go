package v2_test

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/fsamin/go-dump"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	v2 "github.com/ovh/cds/sdk/exportentities/v2"
)

func TestWorkflow_checkDependencies(t *testing.T) {
	type fields struct {
		Name          string
		Description   string
		Version       string
		Workflow      map[string]v2.NodeEntry
		Hooks         map[string][]v2.HookEntry
		Permissions   map[string]int
		HistoryLength int64
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Complex Workflow with a dependency should not raise an error",
			fields: fields{
				Name:    "myWorkflow",
				Version: exportentities.WorkflowVersion2,
				Workflow: map[string]v2.NodeEntry{
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
				Name:    "myWorkflow",
				Version: exportentities.WorkflowVersion2,
				Workflow: map[string]v2.NodeEntry{
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
			w := v2.Workflow{
				Name:        tt.fields.Name,
				Description: tt.fields.Description,
				Version:     tt.fields.Version,
				Workflow:    tt.fields.Workflow,
				Hooks:       tt.fields.Hooks,
				Permissions: tt.fields.Permissions,
			}
			if err := w.CheckDependencies(); (err != nil) != tt.wantErr {
				t.Errorf("Workflow.checkDependencies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkflow_checkValidity(t *testing.T) {
	type fields struct {
		Name        string
		Version     string
		Workflow    map[string]v2.NodeEntry
		Hooks       map[string][]v2.HookEntry
		Permissions map[string]int
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should not raise an error",
			fields: fields{
				Name:    "myWorkflow",
				Version: exportentities.WorkflowVersion2,
				Workflow: map[string]v2.NodeEntry{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := v2.Workflow{
				Name:        tt.fields.Name,
				Version:     tt.fields.Version,
				Workflow:    tt.fields.Workflow,
				Hooks:       tt.fields.Hooks,
				Permissions: tt.fields.Permissions,
			}
			if err := w.CheckValidity(); (err != nil) != tt.wantErr {
				t.Errorf("Workflow.checkValidity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkflow_GetWorkflow(t *testing.T) {
	true := true
	type fields struct {
		Name            string
		Description     string
		Version         string
		Workflow        map[string]v2.NodeEntry
		Hooks           map[string][]v2.HookEntry
		Permissions     map[string]int
		HistoryLength   int64
		RetentionPolicy string
	}
	tsts := []struct {
		name    string
		fields  fields
		want    sdk.Workflow
		wantErr bool
	}{
		// hook conditions
		{
			name: "Workflow with multiple nodes should display hook conditions",
			fields: fields{
				Name:    "myWorkflow",
				Version: exportentities.WorkflowVersion2,
				Workflow: map[string]v2.NodeEntry{
					"root": {
						PipelineName: "pipeline-root",
					},
					"child": {
						PipelineName: "pipeline-child",
						DependsOn:    []string{"root"},
						OneAtATime:   &v2.True,
					},
				},
				Hooks: map[string][]v2.HookEntry{
					"root": {{
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
				RetentionPolicy: "return false",
			},
			wantErr: false,
			want: sdk.Workflow{
				Name:          "myWorkflow",
				HistoryLength: sdk.DefaultHistoryLength,
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
				RetentionPolicy: "return false",
			},
		},
		// root(pipeline-root) -> child(pipeline-child)
		{
			name: "Complexe workflow without joins and mutex should not raise an error",
			fields: fields{
				Name:    "myWorkflow",
				Version: exportentities.WorkflowVersion2,
				Workflow: map[string]v2.NodeEntry{
					"root": {
						PipelineName: "pipeline-root",
					},
					"child": {
						PipelineName: "pipeline-child",
						DependsOn:    []string{"root"},
						OneAtATime:   &v2.True,
					},
				},
			},
			wantErr: false,
			want: sdk.Workflow{
				Name:          "myWorkflow",
				HistoryLength: sdk.DefaultHistoryLength,
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
				Name:    "myWorkflow",
				Version: exportentities.WorkflowVersion2,
				Workflow: map[string]v2.NodeEntry{
					"root": {
						PipelineName: "pipeline-root",
					},
					"child": {
						PipelineName: "pipeline-child",
						DependsOn:    []string{"root"},
						Payload: map[string]interface{}{
							"test": "content",
						},
						OneAtATime: &v2.True,
					},
				},
			},
			wantErr: true,
		},
		// root(pipeline-root) -> child(pipeline-child)
		{
			name: "Complexe workflow unordered without joins should not raise an error",
			fields: fields{
				Name:    "myWorkflow",
				Version: exportentities.WorkflowVersion2,
				Workflow: map[string]v2.NodeEntry{
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
				Name:          "myWorkflow",
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
				Name:    "myWorkflow",
				Version: exportentities.WorkflowVersion2,
				Workflow: map[string]v2.NodeEntry{
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
				Name:          "myWorkflow",
				HistoryLength: sdk.DefaultHistoryLength,
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
				Name:    "myWorkflow",
				Version: exportentities.WorkflowVersion2,
				Workflow: map[string]v2.NodeEntry{
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
				Hooks: map[string][]v2.HookEntry{
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
				Name:          "myWorkflow",
				HistoryLength: sdk.DefaultHistoryLength,
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
			name: "Root and a outgoing hook should not raise an error",
			fields: fields{
				Name:    "myWorkflow",
				Version: exportentities.WorkflowVersion2,
				Workflow: map[string]v2.NodeEntry{
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
				Name:          "myWorkflow",
				HistoryLength: sdk.DefaultHistoryLength,
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
	}

	for _, tt := range tsts {
		t.Run(tt.name, func(t *testing.T) {
			w := v2.Workflow{
				Name:            tt.fields.Name,
				Description:     tt.fields.Description,
				Version:         tt.fields.Version,
				Workflow:        tt.fields.Workflow,
				Hooks:           tt.fields.Hooks,
				Permissions:     tt.fields.Permissions,
				HistoryLength:   &tt.fields.HistoryLength,
				RetentionPolicy: tt.fields.RetentionPolicy,
			}
			got, err := exportentities.ParseWorkflow(w)
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

func TestFromYAMLToYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "Retention policy",
			yaml: `name: retention
version: v2.0
workflow:
  1_start:
    conditions:
      check:
      - variable: git.branch
        operator: eq
        value: master
    pipeline: test
retention_policy: return false
`,
		},
		{
			name: "1_start -> 2_webhook -> 3_after_webhook -> 4_fork_before_end -> 5_end",
			yaml: `name: test1
version: v2.0
workflow:
  1_start:
    conditions:
      check:
      - variable: git.branch
        operator: eq
        value: master
    pipeline: test
  2_webHook:
    depends_on:
    - 1_start
    trigger: WebHook
    config:
      URL: a
      method: POST
      payload: '{}'
  3_after_webhook:
    depends_on:
    - 2_webHook
    when:
    - success
    pipeline: test
  4_fork_before_end:
    depends_on:
    - 3_after_webhook
  5_end:
    depends_on:
    - 4_fork_before_end
    when:
    - success
    pipeline: test
`,
		}, {
			name: "test with outgoing hooks",
			yaml: `name: DDOS
version: v2.0
workflow:
  1_test-outgoing-hooks:
    pipeline: DDOS-me
    payload:
      plip: plop
  2_WebHook:
    depends_on:
    - 1_test-outgoing-hooks
    trigger: WebHook
    config:
      URL: a
      method: POST
      payload: '{}'
  2_after:
    depends_on:
    - 2_WebHook
    when:
    - success
    pipeline: DDOS-me
  3_Workflow:
    depends_on:
    - 1_test-outgoing-hooks
    trigger: Workflow
    config:
      target_hook: bd9ca90e-02e8-4559-9eca-9c56f1518945
      target_project: FSAMIN
      target_workflow: blabla
  3_after:
    depends_on:
    - 3_Workflow
    when:
    - success
    pipeline: DDOS-me
metadata:
  default_tags: git.branch,git.author
`,
		}, {
			name: "tests with outgoing hooks with a join",
			yaml: `name: DDOS
version: v2.0
workflow:
  1_test-outgoing-hooks:
    pipeline: DDOS-me
    payload:
      plip: plop
  2_WebHook:
    depends_on:
    - 1_test-outgoing-hooks
    trigger: WebHook
    config:
      URL: a
      method: POST
      payload: '{}'
  2_after:
    depends_on:
    - 2_WebHook
    when:
    - success
    pipeline: DDOS-me
  3_Workflow:
    depends_on:
    - 1_test-outgoing-hooks
    trigger: Workflow
    config:
      target_hook: bd9ca90e-02e8-4559-9eca-9c56f1518945
      target_project: FSAMIN
      target_workflow: blabla
  3_after:
    depends_on:
    - 3_Workflow
    when:
    - success
    pipeline: DDOS-me
  4_end:
    depends_on:
    - 2_after
    - 3_after
    when:
    - success
    pipeline: DDOS-me
metadata:
  default_tags: git.branch,git.author
`,
		}, {
			name: "test with outgoing hooks, a join, and a fork",
			yaml: `name: DDOS
version: v2.0
workflow:
  1_test-outgoing-hooks:
    pipeline: DDOS-me
    payload:
      plip: plop
  2_WebHook:
    depends_on:
    - 1_test-outgoing-hooks
    trigger: WebHook
    config:
      URL: a
      method: POST
      payload: '{}'
  2_after:
    depends_on:
    - 2_WebHook
    when:
    - success
    pipeline: DDOS-me
  3_Workflow:
    depends_on:
    - 1_test-outgoing-hooks
    trigger: Workflow
    config:
      target_hook: bd9ca90e-02e8-4559-9eca-9c56f1518945
      target_project: FSAMIN
      target_workflow: blabla
  3_after:
    depends_on:
    - 3_Workflow
    when:
    - success
    pipeline: DDOS-me
  4_end:
    depends_on:
    - 2_after
    - 3_after
    when:
    - success
    pipeline: DDOS-me
  "6_1":
    depends_on:
    - fork_1
    when:
    - success
    pipeline: DDOS-me
  "6_2":
    depends_on:
    - fork_1
    when:
    - success
    pipeline: DDOS-me
  fork_1:
    depends_on:
    - 4_end
metadata:
  default_tags: git.branch,git.author
`,
		}, {
			name: "pipeline with two hooks",
			yaml: `name: test3
version: v2.0
workflow:
  1_start:
    pipeline: test
  2_webHook:
    depends_on:
    - 1_start
    trigger: WebHook
    config:
      URL: a
      method: POST
      payload: '{}'
  3_after_webhook:
    depends_on:
    - 2_webHook
    when:
    - success
    pipeline: test
  4_fork_before_end:
    depends_on:
    - 3_after_webhook
  5_end:
    depends_on:
    - 4_fork_before_end
    when:
    - success
    pipeline: test
hooks:
  1_start:
  - type: Scheduler
  - type: WebHook
    config:
      method: POST
`,
		}, {
			name: "Join with condition",
			yaml: `name: joins
version: v2.0
workflow:
  aa:
    pipeline: aa
    application: azer
    environment: aa
    parameters:
      aze: '{{.cds.proj.aa}}'
  aa_2:
    depends_on:
    - aa
    when:
    - success
    pipeline: aa
  aa_3:
    depends_on:
    - join
    when:
    - success
    pipeline: aa
  aa_4:
    depends_on:
    - join
    when:
    - success
    pipeline: aa
  aa_5:
    depends_on:
    - aa_3
    - aa_4
    when:
    - success
    pipeline: aa
  join:
    depends_on:
    - aa
    - aa_2
    when:
    - manual
`,
		},
		{
			name: "Workflow with mutex pipeline child",
			yaml: `name: mymutex
version: v2.0
workflow:
  env:
    pipeline: env
    one_at_a_time: true
  env_2:
    depends_on:
    - env
    when:
    - success
    pipeline: env
    one_at_a_time: true
`,
		},
		{
			name: "Workflow no declared joins",
			yaml: `name: nojoins
version: v2.0
workflow:
  p1:
    pipeline: env
  p5:
    depends_on:
    - p21
    - p31
    - p41
    pipeline: env
  p6:
    depends_on:
    - p21
    - p31
    - p41
    - p42
    pipeline: env
  p7:
    depends_on:
    - p21
    - p31
    - p42
    pipeline: env
  p21:
    depends_on:
    - p1
    pipeline: env
  p22:
    depends_on:
    - p1
    pipeline: env
  p31:
    depends_on:
    - p1
    pipeline: env
  p32:
    depends_on:
    - p1
    pipeline: env
  p41:
    depends_on:
    - p1
    pipeline: env
  p42:
    depends_on:
    - p1
    pipeline: env
`,
		},
	}
	for _, tst := range tests {
		t.Run(tst.name, func(t *testing.T) {
			yamlWorkflow, err := exportentities.UnmarshalWorkflow([]byte(tst.yaml), exportentities.FormatYAML)
			if err != nil {
				if !tst.wantErr {
					t.Error("Unmarshal raised an error", err)
					return
				}
			}
			if tst.wantErr {
				t.Error("Unmarshal should return an error but it doesn't")
				return
			}
			w, err := exportentities.ParseWorkflow(yamlWorkflow)
			if err != nil {
				if !tst.wantErr {
					t.Error("GetWorkflow raised an error", err)
					return
				}
			}
			if tst.wantErr {
				t.Error("GetWorkflow should return an error but it doesn't")
				return
			}

			// Set the hook and outgoing hook models properly before export all the things
			w.VisitNode(func(n *sdk.Node, w *sdk.Workflow) {
				for i := range n.Hooks {
					for _, m := range sdk.BuiltinHookModels {
						if n.Hooks[i].HookModelName == m.Name {
							break
						}
					}
				}
				if n.OutGoingHookContext != nil {
					for _, m := range sdk.BuiltinOutgoingHookModels {
						if n.OutGoingHookContext.HookModelName == m.Name {
							n.OutGoingHookContext.HookModelID = m.ID
							break
						}
					}
				}
			})
			exportedWorkflow, err := exportentities.NewWorkflow(context.TODO(), *w)
			if err != nil {
				if !tst.wantErr {
					t.Error("NewWorkflow raised an error", err)
					return
				}
			}
			if tst.wantErr {
				t.Error("NewWorkflow should return an error but it doesn't")
				return
			}
			b, err := yaml.Marshal(exportedWorkflow)
			if err != nil {
				if !tst.wantErr {
					t.Error("Marshal raised an error", err)
					return
				}
			}
			if tst.wantErr {
				t.Error("Marshal should return an error but it doesn't")
				return
			}
			assert.Equal(t, tst.yaml, string(b))
		})
	}
}

func TestWOrkflowWith2RootsShouldFail(t *testing.T) {
	input := `name: qa-infra
version: v2.0
workflow:
    qa-infra-lint:
      pipeline: qa-infra-lint
      application: qa-infra
    qa-infra-build:
      pipeline: qa-infra
      application: qa-infra
      # Execute only when a user triggered the workflow through the UI
      conditions:
        script: return cds_manual == "true"
      one_at_a_time: true
metadata:
    default_tags: git.branch,git.author 
notifications:
- type: vcs
  settings:
  on_success: always`

	yamlWorkflow, err := exportentities.UnmarshalWorkflow([]byte(input), exportentities.FormatYAML)
	require.NoError(t, err)

	t.Logf("yamlWorkflow> %+v", yamlWorkflow)
	_, err = exportentities.ParseWorkflow(yamlWorkflow)
	require.Error(t, err)
	t.Log(err)

}
