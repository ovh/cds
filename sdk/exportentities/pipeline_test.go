package exportentities

import (
	"encoding/json"
	"fmt"
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
			Name:        "MyPipeline t1_1",
			Description: "my description",
			Type:        sdk.BuildPipeline,
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
										Parameters: []sdk.Parameter{
											{
												Name:  "script",
												Type:  sdk.TextParameter,
												Value: "echo lol\n#This is a script",
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
									{
										Type:           sdk.BuiltinAction,
										Name:           sdk.JUnitAction,
										Enabled:        true,
										AlwaysExecuted: true,
										Optional:       false,
										Parameters: []sdk.Parameter{
											{
												Name:  "path",
												Type:  sdk.StringParameter,
												Value: "path",
											},
										},
									},
									{
										Type:    sdk.BuiltinAction,
										Name:    sdk.ArtifactDownload,
										Enabled: true,
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

										Type:    sdk.BuiltinAction,
										Name:    sdk.ArtifactUpload,
										Enabled: true,
										Parameters: []sdk.Parameter{
											{
												Name:  "path",
												Type:  sdk.StringParameter,
												Value: "path1",
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
			Name:        "MyPipeline",
			Description: "my description",
			Type:        "build",
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

									Type:           sdk.BuiltinAction,
									Name:           sdk.ScriptAction,
									Enabled:        false,
									AlwaysExecuted: true,
									Optional:       false,
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

									Type:           sdk.BuiltinAction,
									Name:           sdk.ScriptAction,
									Enabled:        false,
									AlwaysExecuted: true,
									Optional:       false,
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
									Type:           sdk.BuiltinAction,
									Name:           sdk.ScriptAction,
									Enabled:        false,
									AlwaysExecuted: true,
									Optional:       false,
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
		p := NewPipeline(tc.arg, false)
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
		p := NewPipeline(tc.arg, false)
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
		t.Log(tc.name)
		p := NewPipeline(tc.arg, true)

		b, err := Marshal(p, FormatYAML)
		test.NoError(t, err)

		importedP := Pipeline{}

		test.NoError(t, yaml.Unmarshal(b, &importedP))
		transformedP, err := importedP.Pipeline()

		test.NoError(t, err)

		t.Log(string(b))

		assert.Equal(t, tc.arg.Name, transformedP.Name)
		assert.Equal(t, tc.arg.Description, transformedP.Description)
		assert.Equal(t, tc.arg.Type, transformedP.Type)
		test.EqualValuesWithoutOrder(t, tc.arg.GroupPermission, transformedP.GroupPermission)
		test.EqualValuesWithoutOrder(t, tc.arg.Parameter, transformedP.Parameter)
		for _, stage := range tc.arg.Stages {
			var stageFound bool

			for _, s1 := range transformedP.Stages {
				if stage.Name != s1.Name {
					continue
				}

				stageFound = true

				assert.Equal(t, stage.BuildOrder, s1.BuildOrder, "Build order does not match")
				assert.Equal(t, stage.Enabled, s1.Enabled, "Enabled does not match")
				test.EqualValuesWithoutOrder(t, stage.Prerequisites, s1.Prerequisites)

				for _, j := range stage.Jobs {
					var jobFound bool
					for _, j1 := range s1.Jobs {
						if j.Action.Name != j1.Action.Name {
							continue
						}
						jobFound = true

						assert.Equal(t, j.Enabled, j1.Enabled)
						assert.Equal(t, j.Action.Name, j1.Action.Name)
						assert.Equal(t, j.Enabled, j1.Action.Enabled)
						assert.Equal(t, j.Action.AlwaysExecuted, j1.Action.AlwaysExecuted)
						assert.Equal(t, j.Action.Optional, j1.Action.Optional)

						for i, s := range j.Action.Actions {
							s1 := j1.Action.Actions[i]
							if s.Name == s1.Name {
								assert.Equal(t, s.Enabled, s1.Enabled, s.Name, j1.Action.Name+"/"+s1.Name)
								assert.Equal(t, s.AlwaysExecuted, s1.AlwaysExecuted, j1.Action.Name+"/"+s1.Name)
								assert.Equal(t, s.Optional, s1.Optional, j1.Action.Name+"/"+s1.Name)
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

func Test_ImportPipelineWithRequirements(t *testing.T) {
	in := `name: build-all-images
type: build
requirements:
- hostname: buildbot_image
- binary: git
steps:
- script: |-
    #!/bin/bash

    echo "I'm just a decoy allowing you to rebuild the images."

    exit 0;
`

	payload := &Pipeline{}
	test.NoError(t, yaml.Unmarshal([]byte(in), payload))

	p, err := payload.Pipeline()
	test.NoError(t, err)

	assert.Len(t, p.Stages[0].Jobs[0].Action.Requirements, 2)

}

func Test_ImportPipelineWithGitClone(t *testing.T) {
	in := `name: build-all-images
requirements:
- binary: git
steps:
- gitClone:
    branch: '{{.git.branch}}'
    commit: '{{.git.hash}}'
    directory: '{{.cds.workspace}}'
    password: ""
    privateKey: '{{.cds.app.key}}'
    url: '{{.git.http_url}}'
    user: ""
- artifactUpload:
    path: arti.tar.gz
    tag: '{{.cds.version}}'
- artifactUpload: arti.tar.gz
`

	payload := &Pipeline{}
	test.NoError(t, yaml.Unmarshal([]byte(in), payload))

	p, err := payload.Pipeline()
	test.NoError(t, err)

	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions, 3)
	assert.Equal(t, sdk.GitCloneAction, p.Stages[0].Jobs[0].Action.Actions[0].Name)
	assert.Equal(t, sdk.ArtifactUpload, p.Stages[0].Jobs[0].Action.Actions[1].Name)
	assert.Equal(t, sdk.ArtifactUpload, p.Stages[0].Jobs[0].Action.Actions[2].Name)
	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions[0].Parameters, 7)
	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions[1].Parameters, 2)
	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions[2].Parameters, 1)
}

func Test_ImportPipelineWithCheckout(t *testing.T) {
	in := `name: build-all-images
steps:
- checkout: '.'
`

	payload := &Pipeline{}
	test.NoError(t, yaml.Unmarshal([]byte(in), payload))

	p, err := payload.Pipeline()
	test.NoError(t, err)

	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions, 1)
	assert.Equal(t, sdk.CheckoutApplicationAction, p.Stages[0].Jobs[0].Action.Actions[0].Name)
	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions[0].Parameters, 1)
}

func Test_IsFlagged(t *testing.T) {
	testc := []struct {
		flag     string
		step     Step
		expected bool
	}{
		{
			flag:     "enabled",
			step:     Step{"enabled": true},
			expected: true,
		},
		{
			flag:     "enabled",
			step:     Step{"enabled": false, "optional": true},
			expected: false,
		},
		{
			flag:     "optional",
			step:     Step{"optional": true},
			expected: true,
		},
		{
			flag:     "always_executed",
			step:     Step{"optional": false},
			expected: false,
		},
		{
			flag:     "always_executed",
			step:     Step{"optional": false, "always_executed": true},
			expected: true,
		},
		{
			flag:     "optional",
			step:     Step{"always_executed": true},
			expected: false,
		},
		{
			flag:     "always_executed",
			step:     Step{"always_executed": true, "enabled": false, "optional": false},
			expected: true,
		},
	}

	for _, tc := range testc {
		resp, err := tc.step.IsFlagged(tc.flag)
		test.NoError(t, err, fmt.Sprintf("Flag %s should not return an error in this step", tc.flag))
		assert.Equal(t, tc.expected, resp, fmt.Sprintf("Flag %s have bad value in this step", tc.flag))
	}

}

func TestExportPipelineV1_YAML(t *testing.T) {
	for _, tc := range testcases {
		p := NewPipelineV1(tc.arg, false)
		b, err := Marshal(p, FormatYAML)
		test.NoError(t, err)
		t.Log("\n" + string(b))

		p1 := PipelineV1{}
		test.NoError(t, yaml.Unmarshal(b, &p1))

		test.Equal(t, p, p1)
	}
}

func TestExportPipelineV1_JSON(t *testing.T) {
	for _, tc := range testcases {
		p := NewPipelineV1(tc.arg, false)
		b, err := Marshal(p, FormatJSON)
		test.NoError(t, err)
		t.Log("\n" + string(b))

		p1 := PipelineV1{}
		test.NoError(t, json.Unmarshal(b, &p1))

		test.Equal(t, p, p1)
	}
}

func TestExportAndImportPipelineV1_YAML(t *testing.T) {
	for _, tc := range testcases {
		t.Log(tc.name)
		p := NewPipelineV1(tc.arg, true)

		b, err := Marshal(p, FormatYAML)
		test.NoError(t, err)

		importedP := PipelineV1{}

		test.NoError(t, yaml.Unmarshal(b, &importedP))
		transformedP, err := importedP.Pipeline()

		test.NoError(t, err)

		t.Log(string(b))

		assert.Equal(t, tc.arg.Name, transformedP.Name)
		assert.Equal(t, tc.arg.Type, transformedP.Type)
		test.EqualValuesWithoutOrder(t, tc.arg.GroupPermission, transformedP.GroupPermission)
		test.EqualValuesWithoutOrder(t, tc.arg.Parameter, transformedP.Parameter)
		for _, stage := range tc.arg.Stages {
			var stageFound bool

			for _, s1 := range transformedP.Stages {
				if stage.Name != s1.Name {
					continue
				}

				stageFound = true

				assert.Equal(t, stage.BuildOrder, s1.BuildOrder, "Build order does not match")
				assert.Equal(t, stage.Enabled, s1.Enabled, "Enabled does not match")
				test.EqualValuesWithoutOrder(t, stage.Prerequisites, s1.Prerequisites)

				for _, j := range stage.Jobs {
					var jobFound bool
					for _, j1 := range s1.Jobs {
						if j.Action.Name != j1.Action.Name {
							continue
						}
						jobFound = true

						assert.Equal(t, j.Enabled, j1.Enabled)
						assert.Equal(t, j.Action.Name, j1.Action.Name)
						assert.Equal(t, j.Enabled, j1.Action.Enabled)
						assert.Equal(t, j.Action.AlwaysExecuted, j1.Action.AlwaysExecuted)
						assert.Equal(t, j.Action.Optional, j1.Action.Optional)

						for i, s := range j.Action.Actions {
							s1 := j1.Action.Actions[i]
							if s.Name == s1.Name {
								assert.Equal(t, s.Enabled, s1.Enabled, s.Name, j1.Action.Name+"/"+s1.Name)
								assert.Equal(t, s.AlwaysExecuted, s1.AlwaysExecuted, j1.Action.Name+"/"+s1.Name)
								assert.Equal(t, s.Optional, s1.Optional, j1.Action.Name+"/"+s1.Name)
								test.EqualValuesWithoutOrder(t, s.Parameters, s1.Parameters)
							}
						}
					}
					assert.True(t, jobFound, "Job not found")
				}

			}
			if len(tc.arg.Stages) > 1 {
				assert.True(t, stageFound, "Stage not found")
			}
		}
	}
}
