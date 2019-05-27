package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Script action definition.
var Script = Manifest{
	Action: sdk.Action{
		Name:        sdk.ScriptAction,
		Description: `This action executes a given script with a given interpreter.`,
		Parameters: []sdk.Parameter{
			{
				Name: "script",
				Description: `Content of your script.
You can put #!/bin/bash, or #!/bin/perl at first line.
Make sure that the binary used is in
the pre-requisites of action.`,
				Type: sdk.TextParameter,
			},
		},
	},
	Example: exportentities.PipelineV1{
		Version: exportentities.PipelineVersion1,
		Name:    "Pipeline1",
		Stages:  []string{"Stage1"},
		Jobs: []exportentities.Job{{
			Name:  "Job1",
			Stage: "Stage1",
			Steps: []exportentities.Step{
				{
					Script: []string{
						"#!/bin/sh",
						"echo \"{{.cds.application}}\"",
					},
				},
			},
		}},
	},
}
