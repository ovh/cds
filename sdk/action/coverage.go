package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Coverage action definition.
var Coverage = Manifest{
	Action: sdk.Action{
		Name: sdk.CoverageAction,
		Description: `CDS Builtin Action.
Upload you coverage file to CDS as a coverage run result.
`,
		Parameters: []sdk.Parameter{
			{
				Name:        "path",
				Description: `Path of the coverage report file.`,
				Type:        sdk.StringParameter,
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
					Coverage: &exportentities.StepCoverage{
						Path: "./coverage.xml",
					},
				},
			},
		}},
	},
}
