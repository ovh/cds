package exportentities_test

import (
	"encoding/json"
	"testing"

	"github.com/ovh/cds/sdk/exportentities"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

type pipelineTestCase struct {
	name     string
	arg      sdk.Pipeline
	expected exportentities.PipelineV1
}

var (
	t1_1 = pipelineTestCase{
		name: "Pipeline with 1 stage and 1 job",
		arg: sdk.Pipeline{
			Name:        "MyPipeline t1_1",
			Description: "my description",
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
								Requirements: []sdk.Requirement{
									{
										Name:  sdk.OSArchRequirement,
										Type:  sdk.OSArchRequirement,
										Value: "freebsd/amd64",
									},
									{
										Name:  sdk.RegionRequirement,
										Type:  sdk.RegionRequirement,
										Value: "graxyz",
									},
								},
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
											{
												Name:  "pattern",
												Type:  sdk.StringParameter,
												Value: "thepattern",
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
									{
										Type:    sdk.BuiltinAction,
										Name:    sdk.CoverageAction,
										Enabled: true,
										Parameters: []sdk.Parameter{
											{
												Name:  "path",
												Type:  sdk.StringParameter,
												Value: "lcov.info",
											},
											{
												Name:  "format",
												Type:  sdk.StringParameter,
												Value: "lcov",
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
		expected: exportentities.PipelineV1{
			Name:        "MyPipeline",
			Description: "my description",
		},
	}

	t1_2 = pipelineTestCase{
		name: "Pipeline with 1 stage and 2 jobs",
		arg: sdk.Pipeline{
			Name: "MyPipeline t1_2",
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
		expected: exportentities.PipelineV1{
			Name: "MyPipeline",
		},
	}

	t2_2 = pipelineTestCase{
		name: "Pipeline with 2 stages and 2 jobs",
		arg: sdk.Pipeline{
			Name: "MyPipeline t2_2",
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
					Conditions: sdk.WorkflowNodeConditions{
						PlainConditions: []sdk.WorkflowNodeCondition{
							{
								Variable: "param1",
								Operator: "regex",
								Value:    "value1",
							},
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
		expected: exportentities.PipelineV1{
			Name: "MyPipeline",
		},
	}

	tRealease = pipelineTestCase{
		name: "Pipeline with 1 stages and 1 release job",
		arg: sdk.Pipeline{
			Name: "MyPipeline tRelease",
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
									Name:    sdk.ReleaseAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "artifacts",
											Type:  sdk.StringParameter,
											Value: ".*",
										},
										{
											Name:  "releaseNote",
											Type:  sdk.StringParameter,
											Value: "my release",
										},
										{
											Name:  "srcMaturity",
											Type:  sdk.StringParameter,
											Value: "staging",
										},
										{
											Name:  "destMaturity",
											Type:  sdk.StringParameter,
											Value: "rc",
										},
									},
								},
							},
						},
					}},
				},
			},
		},
	}

	tPromote = pipelineTestCase{
		name: "Pipeline with 1 stages and 1 release job",
		arg: sdk.Pipeline{
			Name: "MyPipeline tRelease",
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
									Name:    sdk.PromoteAction,
									Enabled: true,
									Parameters: []sdk.Parameter{
										{
											Name:  "artifacts",
											Type:  sdk.StringParameter,
											Value: ".*",
										},
										{
											Name:  "srcMaturity",
											Type:  sdk.StringParameter,
											Value: "snapshot",
										},
										{
											Name:  "destMaturity",
											Type:  sdk.StringParameter,
											Value: "staging",
										},
									},
								},
							},
						},
					}},
				},
			},
		},
	}
	testcases = []pipelineTestCase{t1_1, t1_2, t2_2, tRealease}
)

func TestExportPipeline_YAML(t *testing.T) {
	for _, tc := range testcases {
		p := exportentities.NewPipelineV1(tc.arg)
		b, err := exportentities.Marshal(p, exportentities.FormatYAML)
		test.NoError(t, err)
		t.Log("\n" + string(b))

		p1 := exportentities.PipelineV1{}
		test.NoError(t, yaml.Unmarshal(b, &p1))

		test.Equal(t, p, p1)
	}
}

func TestExportPipeline_JSON(t *testing.T) {
	for _, tc := range testcases {
		p := exportentities.NewPipelineV1(tc.arg)
		b, err := exportentities.Marshal(p, exportentities.FormatJSON)
		test.NoError(t, err)
		t.Log("\n" + string(b))

		p1 := exportentities.PipelineV1{}
		test.NoError(t, json.Unmarshal(b, &p1))

		test.Equal(t, p, p1)
	}
}

func TestExportAndImportPipeline_YAML(t *testing.T) {
	for _, tc := range testcases {
		t.Log(tc.name)
		p := exportentities.NewPipelineV1(tc.arg)

		b, err := exportentities.Marshal(p, exportentities.FormatYAML)
		test.NoError(t, err)

		importedP := exportentities.PipelineV1{}

		test.NoError(t, yaml.Unmarshal(b, &importedP))
		transformedP, err := importedP.Pipeline()

		test.NoError(t, err)

		t.Log(string(b))

		assert.Equal(t, tc.arg.Name, transformedP.Name)
		assert.Equal(t, tc.arg.Description, transformedP.Description)
		test.EqualValuesWithoutOrder(t, tc.arg.Parameter, transformedP.Parameter)
		test.Equal(t, len(tc.arg.Stages), len(transformedP.Stages))
		for _, stage := range tc.arg.Stages {
			var stageFound bool

			for _, s1 := range transformedP.Stages {
				if stage.Name != s1.Name && len(tc.arg.Stages) != 1 {
					continue
				}

				stageFound = true

				assert.Equal(t, stage.BuildOrder, s1.BuildOrder, "Build order does not match")
				assert.Equal(t, stage.Enabled, s1.Enabled, "Enabled does not match")
				test.EqualValuesWithoutOrder(t, stage.Conditions.PlainConditions, s1.Conditions.PlainConditions)

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
jobs:
- job: build
  requirements:
  - hostname: buildbot_image
  - binary: git
  steps:
  - script: |-
      #!/bin/bash

      echo "I'm just a decoy allowing you to rebuild the images."

      exit 0;
`

	payload := &exportentities.PipelineV1{}
	test.NoError(t, yaml.Unmarshal([]byte(in), payload))

	p, err := payload.Pipeline()
	test.NoError(t, err)

	assert.Len(t, p.Stages[0].Jobs[0].Action.Requirements, 2)
}

func Test_ImportPipelineWithPromote(t *testing.T) {
	in := `version: v1.0
name: push-artifact
stages:
- Stage 1
- Stage 2
- Stage 3
- Stage 4
jobs:
- job: Push-artifact
  stage: Stage 1
  steps:
  - script:
    - env > env.txt
  - artifactUpload:
      path: env.txt
      tag: '{{.cds.version}}'
- job: Release-to-staging
  stage: Stage 2
  steps:
  - promote:
      artifacts: .*
      srcMaturity: snapshot
      destMaturity: staging
- job: Release-to-rc
  stage: Stage 3
  steps:
  - release:
      artifacts: .*
      srcMaturity: staging
      destMaturity: rc
- job: Release-to-release
  stage: Stage 4
  steps:
  - release:
      artifacts: .*
      srcMaturity: rc
      destMaturity: release
`

	payload := &exportentities.PipelineV1{}
	test.NoError(t, yaml.Unmarshal([]byte(in), payload))

	p, err := payload.Pipeline()
	test.NoError(t, err)

	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions, 2)
	assert.Len(t, p.Stages[1].Jobs[0].Action.Actions, 1)
	assert.Len(t, p.Stages[2].Jobs[0].Action.Actions, 1)
	assert.Len(t, p.Stages[3].Jobs[0].Action.Actions, 1)
}

func Test_ImportPipelineWithGitClone(t *testing.T) {
	in := `name: build-all-images
jobs:
- job: build
  requirements:
  - binary: git
  - os-archicture: freebsd/amd64
  - region: graxyz
  steps:
  - gitClone:
      branch: '{{.git.branch}}'
      commit: '{{.git.hash}}'
      directory: '{{.cds.workspace}}'
      password: ""
      privateKey: '{{.cds.app.key}}'
      url: '{{.git.http_url}}'
      user: ""
      depth: '12'
  - artifactUpload:
      path: arti.tar.gz
      tag: '{{.cds.version}}'
`

	payload := &exportentities.PipelineV1{}
	test.NoError(t, yaml.Unmarshal([]byte(in), payload))

	p, err := payload.Pipeline()
	test.NoError(t, err)

	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions, 2)
	assert.Len(t, p.Stages[0].Jobs[0].Action.Requirements, 3)
	assert.Equal(t, sdk.GitCloneAction, p.Stages[0].Jobs[0].Action.Actions[0].Name)
	assert.Equal(t, sdk.ArtifactUpload, p.Stages[0].Jobs[0].Action.Actions[1].Name)
	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions[0].Parameters, 6)
	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions[1].Parameters, 2)
}

func Test_ImportPipelineWithCheckout(t *testing.T) {
	in := `name: build-all-images
jobs:
- job: build
  steps:
  - checkout: '.'
`

	payload := &exportentities.PipelineV1{}
	test.NoError(t, yaml.Unmarshal([]byte(in), payload))

	p, err := payload.Pipeline()
	test.NoError(t, err)

	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions, 1)
	assert.Equal(t, sdk.CheckoutApplicationAction, p.Stages[0].Jobs[0].Action.Actions[0].Name)
	assert.Len(t, p.Stages[0].Jobs[0].Action.Actions[0].Parameters, 1)
}

func Test_ImportPipelineWithOneStageAndRunConditions(t *testing.T) {
	in := `version: v1.0
name: echo
stages:
- Stage 1
options:
  Stage 1:
    conditions:
      check:
      - variable: git.branch
        operator: ne
        value: ""
jobs:
- job: New Job
`

	payload := &exportentities.PipelineV1{}
	test.NoError(t, yaml.Unmarshal([]byte(in), payload))

	p, err := payload.Pipeline()
	test.NoError(t, err)

	assert.Len(t, p.Stages, 1)
}

func Test_ImportPipeline2TimesStage(t *testing.T) {
	in := `version: v1.0
name: echo
stages:
- Stage 1
options:
  Stage 1:
    conditions:
      check:
      - variable: git.branch
        operator: ne
        value: ""
jobs:
- job: New Job
`

	payload := &exportentities.PipelineV1{}
	test.NoError(t, yaml.Unmarshal([]byte(in), payload))

	p, err := payload.Pipeline()
	test.NoError(t, err)

	assert.Len(t, p.Stages, 1)

	in = `version: v1.0
name: echo
stages:
- Stage 0
- Stage 1
jobs:
- job: New Job
  stage: Stage 1
  steps:
  - script:
    - echo "coucou"
- job: New Job
  stage: Stage 0
  steps:
  - script:
    - echo "coucou"
`

	payload = &exportentities.PipelineV1{}
	test.NoError(t, yaml.Unmarshal([]byte(in), payload))

	p, err = payload.Pipeline()
	test.NoError(t, err)

	assert.Len(t, p.Stages, 2)
	assert.Equal(t, 1, p.Stages[0].BuildOrder)
	assert.Equal(t, "Stage 0", p.Stages[0].Name)
	assert.Equal(t, 2, p.Stages[1].BuildOrder)
	assert.Equal(t, "Stage 1", p.Stages[1].Name)
}

func TestExportPipelineV1_YAML(t *testing.T) {
	for _, tc := range testcases {
		p := exportentities.NewPipelineV1(tc.arg)
		b, err := exportentities.Marshal(p, exportentities.FormatYAML)
		test.NoError(t, err)
		t.Log("\n" + string(b))

		p1 := exportentities.PipelineV1{}
		test.NoError(t, yaml.Unmarshal(b, &p1))

		test.Equal(t, p, p1)
	}
}

func TestExportPipelineV1_JSON(t *testing.T) {
	for _, tc := range testcases {
		p := exportentities.NewPipelineV1(tc.arg)
		b, err := exportentities.Marshal(p, exportentities.FormatJSON)
		test.NoError(t, err)
		t.Log("\n" + string(b))

		p1 := exportentities.PipelineV1{}
		test.NoError(t, json.Unmarshal(b, &p1))

		test.Equal(t, p, p1)
	}
}

func TestExportAndImportPipelineV1_YAML(t *testing.T) {
	for _, tc := range testcases {
		t.Log(tc.name)
		p := exportentities.NewPipelineV1(tc.arg)

		b, err := exportentities.Marshal(p, exportentities.FormatYAML)
		test.NoError(t, err)

		importedP := exportentities.PipelineV1{}

		test.NoError(t, yaml.Unmarshal(b, &importedP))
		transformedP, err := importedP.Pipeline()

		test.NoError(t, err)

		t.Log(string(b))

		assert.Equal(t, tc.arg.Name, transformedP.Name)
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
				test.EqualValuesWithoutOrder(t, stage.Conditions.PlainConditions, s1.Conditions.PlainConditions)

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
