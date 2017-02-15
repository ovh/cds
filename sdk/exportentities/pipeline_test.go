package exportentities

import (
	"encoding/json"
	"testing"

	"github.com/fsamin/go-dump"
	"github.com/hashicorp/hcl"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

type pipelineTestCase struct {
	name     string
	arg      sdk.Pipeline
	expected Pipeline
}

var (
	t1_1 = pipelineTestCase{
		name: "Pipeline with 1 stage and 1 job",
		arg: sdk.Pipeline{
			Name: "MyPipeline",
			Type: sdk.BuildPipeline,
			Stages: []sdk.Stage{
				{
					BuildOrder: 1,
					Name:       "stage 1",
					Enabled:    true,
					Jobs: []sdk.Job{
						{
							Action: sdk.Action{
								Name:        "Job 1",
								Description: "This is job 1",
								Actions: []sdk.Action{
									{

										Type:    sdk.BuiltinAction,
										Name:    sdk.ScriptAction,
										Enabled: true,
										Final:   true,
										Parameters: []sdk.Parameter{
											{
												Name:  "script",
												Type:  sdk.TextParameter,
												Value: "echo lol",
											},
										},
									},
									{

										Type: sdk.BuiltinAction,
										Name: sdk.ScriptAction,
										Parameters: []sdk.Parameter{
											{
												Name:  "script",
												Type:  sdk.TextParameter,
												Value: "echo lel",
											},
										},
									},
									{

										Type:  sdk.BuiltinAction,
										Name:  sdk.JUnitAction,
										Final: true,
										Parameters: []sdk.Parameter{
											{
												Name:  "path",
												Type:  sdk.StringParameter,
												Value: "path",
											},
										},
									},
									{

										Type: sdk.BuiltinAction,
										Name: sdk.ArtifactDownload,
										Parameters: []sdk.Parameter{
											{
												Name:  "path",
												Type:  sdk.StringParameter,
												Value: "path1",
											},
											{
												Name:  "tag",
												Type:  sdk.StringParameter,
												Value: "tag1",
											},
										},
									},
									{

										Type: sdk.BuiltinAction,
										Name: sdk.ArtifactUpload,
										Parameters: []sdk.Parameter{
											{
												Name:  "path",
												Type:  sdk.StringParameter,
												Value: "path1",
											},
											{
												Name:  "tag",
												Type:  sdk.StringParameter,
												Value: "tag1",
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
		expected: Pipeline{
			Name: "MyPipeline",
			Type: "build",
		},
	}

	t1_2 = pipelineTestCase{
		name: "Pipeline with 1 stage and 2 jobs",
		arg: sdk.Pipeline{
			Name: "MyPipeline",
			Type: sdk.BuildPipeline,
			GroupPermission: []sdk.GroupPermission{
				sdk.GroupPermission{
					Group: sdk.Group{
						Name: "group1",
					},
					Permission: 4,
				},
			},
			Stages: []sdk.Stage{
				{
					BuildOrder: 1,
					Name:       "stage 1",
					Enabled:    true,
					Jobs: []sdk.Job{{
						Action: sdk.Action{
							Name:        "Job 1",
							Description: "This is job 1",
							Actions: []sdk.Action{
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lol",
										},
									},
								},
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lel",
										},
									},
								},
							},
						},
					}, {
						Action: sdk.Action{
							Name:        "Job 2",
							Description: "This is job 2",
							Actions: []sdk.Action{
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lol",
										},
									},
								},
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: false,
									Final:   true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lel",
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
		expected: Pipeline{
			Name: "MyPipeline",
			Type: "build",
		},
	}

	t2_2 = pipelineTestCase{
		name: "Pipeline with 1 stage and 2 jobs",
		arg: sdk.Pipeline{
			Name: "MyPipeline",
			Type: sdk.BuildPipeline,
			GroupPermission: []sdk.GroupPermission{
				sdk.GroupPermission{
					Group: sdk.Group{
						Name: "group1",
					},
					Permission: 4,
				},
			},
			Stages: []sdk.Stage{
				{
					BuildOrder: 1,
					Name:       "stage 1",
					Enabled:    true,
					Jobs: []sdk.Job{{
						Action: sdk.Action{
							Name:        "Job 1",
							Description: "This is job 1",
							Actions: []sdk.Action{
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lol",
										},
									},
								},
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lel",
										},
									},
								},
							},
						},
					}, {
						Action: sdk.Action{
							Name:        "Job 2",
							Description: "This is job 2",
							Actions: []sdk.Action{
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lol",
										},
									},
								},
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: false,
									Final:   true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lel",
										},
									},
								},
							},
						},
					},
					},
				}, {
					BuildOrder: 2,
					Name:       "stage 2",
					Enabled:    true,
					Prerequisites: []sdk.Prerequisite{
						{
							Parameter:     "param1",
							ExpectedValue: "value1",
						},
					},
					Jobs: []sdk.Job{{
						Action: sdk.Action{
							Name:        "Job 1",
							Description: "This is job 1",
							Actions: []sdk.Action{
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lol",
										},
									},
								},
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lel",
										},
									},
								},
							},
						},
					}, {
						Action: sdk.Action{
							Name:        "Job 2",
							Description: "This is job 2",
							Actions: []sdk.Action{
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lol",
										},
									},
								},
								{

									Type:    sdk.BuiltinAction,
									Name:    sdk.ScriptAction,
									Enabled: false,
									Final:   true,
									Parameters: []sdk.Parameter{
										{
											Name:  "script",
											Type:  sdk.TextParameter,
											Value: "echo lel",
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
		expected: Pipeline{
			Name: "MyPipeline",
			Type: "build",
		},
	}

	testcases = []pipelineTestCase{t1_1, t1_2, t2_2}
)

func TestExportImportPipeline_HCL(t *testing.T) {
	t.SkipNow()
	for _, tc := range testcases {
		p := NewPipeline(&tc.arg)
		b, err := Marshal(p, FormatHCL)
		test.NoError(t, err)
		t.Log("\n" + string(b))

		i1 := map[string]interface{}{}
		test.NoError(t, hcl.Unmarshal(b, &i1))
		/*p1, err := decodePipeline(i1)
		test.NoError(t, err)
		t.Logf("%s", p1)

		t.Logf(dump.Sdump(p))
		t.Logf(dump.Sdump(p1))

		m1, err := dump.ToMap(p1)
		test.NoError(t, err)
		m2, err := dump.ToMap(p)
		assert.EqualValues(t, m2, m1)
		test.NoError(t, err)*/
	}
}

func TestExportImportPipeline_YAML(t *testing.T) {
	for _, tc := range testcases {
		p := NewPipeline(&tc.arg)
		b, err := Marshal(p, FormatYAML)
		test.NoError(t, err)
		t.Log("\n" + string(b))

		p1 := Pipeline{}
		test.NoError(t, yaml.Unmarshal(b, &p1))

		m1, _ := dump.ToMap(p1)
		m2, _ := dump.ToMap(p)
		assert.EqualValues(t, m2, m1)
	}
}

func TestExportImportPipeline_JSON(t *testing.T) {
	for _, tc := range testcases {
		p := NewPipeline(&tc.arg)
		b, err := Marshal(p, FormatJSON)
		test.NoError(t, err)
		t.Log("\n" + string(b))

		p1 := Pipeline{}
		test.NoError(t, json.Unmarshal(b, &p1))

		m1, _ := dump.ToMap(p1)
		m2, _ := dump.ToMap(p)
		assert.EqualValues(t, m2, m1)

	}
}
