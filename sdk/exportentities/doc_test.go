package exportentities_test

import (
	"fmt"
	"testing"

	"github.com/ovh/cds/sdk/exportentities"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

func TestPipeline(t *testing.T) {
	/*	exportentities.PipelineV1{
			Version:     exportentities.PipelineVersion1,
			Name:        "My pipeline name",
			Description: "My pipeline description",
			Stages: []string{
				"Stage1",
				"Stage2",
			},
			StageOptions: map[string]exportentities.Stage{
				"Stage1": exportentities.Stage{
					Enabled: &sdk.True,
					Conditions: map[string]string{
						"cds.manual": "true",
					},
				},
			},
			Parameters: map[string]exportentities.ParameterValue{
				"Param1": exportentities.ParameterValue{
					Type:        string(sdk.ParameterTypeBoolean),
					Description: "My first param description",
				},
				"Param2": exportentities.ParameterValue{
					Type:        string(sdk.ParameterTypeString),
					Description: "My second param description",
				},
			},
			Jobs: []exportentities.Job{
				{
					Name:           "Job1",
					Description:    "My first job description",
					Enabled:        &sdk.True,
					Optional:       &sdk.False,
					AlwaysExecuted: &sdk.True,
					Stage:          "Stage1",
					Requirements: []exportentities.Requirement{
						exportentities.Requirement{Binary: "git"},
						exportentities.Requirement{Hostname: "localhost"},
						exportentities.Requirement{Memory: "2048"},
						exportentities.Requirement{Model: "myModel"},
						exportentities.Requirement{Network: "1.1.1.1"},
						exportentities.Requirement{Plugin: "myPlugin"},
						exportentities.Requirement{Service: exportentities.ServiceRequirement{
							Name:  "my-database",
							Value: "postgres",
						}},
					},
					Steps: []exportentities.Step{
						{
							"script": "echo \"hello world\"",
							"script": []string{
								"echo \"my first line\"",
								"echo \"my second line\"",
							},
						},
					},
				},
			},
	  }*/

	scriptOneLine := sdk.Action{
		Type:    sdk.BuiltinAction,
		Name:    sdk.ScriptAction,
		Enabled: true,
		Parameters: []sdk.Parameter{
			{
				Name:  "script",
				Type:  sdk.TextParameter,
				Value: "echo \"hello world\"",
			},
		},
	}

	scriptMultiLine := sdk.Action{
		Type:    sdk.BuiltinAction,
		Name:    sdk.ScriptAction,
		Enabled: true,
		Parameters: []sdk.Parameter{
			{
				Type:  sdk.TextParameter,
				Name:  "script",
				Value: "echo \"first line\"\necho \"second line\"",
			},
		},
	}

	download := sdk.Action{
		Type:    sdk.BuiltinAction,
		Name:    sdk.ArtifactDownload,
		Enabled: true,
		Parameters: []sdk.Parameter{
			{
				Type:  sdk.StringParameter,
				Name:  "pattern",
				Value: "*.zip",
			},
			{
				Type:  sdk.StringParameter,
				Name:  "path",
				Value: "{{.cds.workspace}}",
			},
			{
				Type:  sdk.StringParameter,
				Name:  "tag",
				Value: "{{.cds.version}}",
			},
		},
	}

	upload := sdk.Action{
		Type:    sdk.BuiltinAction,
		Name:    sdk.ArtifactUpload,
		Enabled: true,
		Parameters: []sdk.Parameter{
			{
				Type:  sdk.StringParameter,
				Name:  "path",
				Value: "{{.cds.workspace}}/*.zip",
			},
			{
				Type:  sdk.StringParameter,
				Name:  "tag",
				Value: "{{.cds.version}}",
			},
		},
	}

	p := sdk.Pipeline{
		Name:        "My pipeline name",
		Description: "My pipeline description",
		Parameter: []sdk.Parameter{
			sdk.Parameter{
				Type:        string(sdk.ParameterTypeBoolean),
				Name:        "Param1",
				Description: "My first param description",
				Advanced:    false,
				Value:       "true",
			},
			sdk.Parameter{
				Type:        string(sdk.ParameterTypeString),
				Name:        "Param2",
				Description: "My second param description",
				Advanced:    true,
				Value:       "default2",
			},
		},
		Stages: []sdk.Stage{
			sdk.Stage{
				Name:    "Stage1",
				Enabled: true,
				Jobs: []sdk.Job{
					{
						Enabled: true,
						Action: sdk.Action{
							Name:        "Job 1",
							Description: "This is job 1",
							Enabled:     true,
							Actions: []sdk.Action{
								scriptOneLine,
								scriptMultiLine,
								download,
								upload,
							},
						},
					},
				},
			},
		},
	}

	eP := exportentities.NewPipelineV1(p)
	buf, err := yaml.Marshal(eP)
	assert.NoError(t, err)

	fmt.Println(string(buf))
}
