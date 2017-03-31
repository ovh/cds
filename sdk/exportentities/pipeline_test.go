package exportentities

import (
	"encoding/json"
	"testing"

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
			Name: "MyPipeline t1_1",
			Type: sdk.BuildPipeline,
			Stages: []sdk.Stage{
				{
					BuildOrder: 1,
					Name:       "MyPipeline t1_1",
					Enabled:    true,
					Jobs: []sdk.Job{
						{
							Enabled: true,
							Action: sdk.Action{
								Name: "MyPipeline t1_1",
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
			Name: "MyPipeline t1_2",
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
					Name:       "MyPipeline t1_2",
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
		name: "Pipeline with 2 stages and 2 jobs",
		arg: sdk.Pipeline{
			Name: "MyPipeline t2_2",
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
						Enabled: true,
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
						Enabled: true,
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
						Enabled: true,
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
						Enabled: true,
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

func TestExportPipeline_YAML(t *testing.T) {
	for _, tc := range testcases {
		p := NewPipeline(&tc.arg)
		b, err := Marshal(p, FormatYAML)
		test.NoError(t, err)
		t.Log("\n" + string(b))

		p1 := Pipeline{}
		test.NoError(t, yaml.Unmarshal(b, &p1))

		test.Equal(t, p, p1)
	}
}

func TestExportPipeline_JSON(t *testing.T) {
	for _, tc := range testcases {
		p := NewPipeline(&tc.arg)
		b, err := Marshal(p, FormatJSON)
		test.NoError(t, err)
		t.Log("\n" + string(b))

		p1 := Pipeline{}
		test.NoError(t, json.Unmarshal(b, &p1))

		test.Equal(t, p, p1)
	}
}

func TestExportAndImportPipeline_YAML(t *testing.T) {
	for _, tc := range testcases {
		p := NewPipeline(&tc.arg)

		b, err := Marshal(p, FormatYAML)
		test.NoError(t, err)

		importedP := Pipeline{}
		test.NoError(t, yaml.Unmarshal(b, &importedP))
		transformedP, err := importedP.Pipeline()

		test.NoError(t, err)

		assert.Equal(t, tc.arg.Name, transformedP.Name)
		assert.Equal(t, tc.arg.Type, transformedP.Type)
		test.EqualValuesWithoutOrder(t, tc.arg.GroupPermission, transformedP.GroupPermission)
		test.EqualValuesWithoutOrder(t, tc.arg.Parameter, transformedP.Parameter)
		for _, s := range tc.arg.Stages {
			var stageFound bool
			for _, s1 := range transformedP.Stages {
				if s.Name != s1.Name {
					continue
				}
				stageFound = true

				assert.Equal(t, s.BuildOrder, s1.BuildOrder, "Build order does not match")
				assert.Equal(t, s.Enabled, s1.Enabled, "Enabled does not match")
				test.EqualValuesWithoutOrder(t, s.Prerequisites, s1.Prerequisites)

				for _, j := range s.Jobs {
					var jobFound bool
					for _, j1 := range s1.Jobs {
						if j.Action.Name != j1.Action.Name {
							continue
						}
						jobFound = true

						assert.Equal(t, j.Enabled, j1.Enabled)
						assert.Equal(t, j.Action.Name, j1.Action.Name)
						assert.Equal(t, j.Enabled, j1.Action.Enabled)
						assert.Equal(t, j.Action.Final, j1.Action.Final)

						for i, s := range j.Action.Actions {
							s1 := j1.Action.Actions[i]
							if s.Name == s1.Name {
								assert.Equal(t, s.Enabled, s1.Enabled, s.Name, s1.Name)
								assert.Equal(t, s.Final, s1.Final)
								test.EqualValuesWithoutOrder(t, s.Parameters, s1.Parameters)
							}
						}
					}
					assert.True(t, jobFound, "Job not found")
				}

			}
			assert.True(t, stageFound, "Stage not found")
		}
	}
}
