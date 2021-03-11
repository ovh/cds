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
Parse given file to extract coverage results for lcov, cobertura and clover format.
Then the coverage report is uploaded in CDN.
Coverage report will be linked to the application from the pipeline context for lcov, cobertura and clover format.
You will be able to see the coverage history in the application home page for lcov, cobertura and clover format.
`,
		Parameters: []sdk.Parameter{
			{
				Name:        "format",
				Description: `Coverage report format.`,
				Type:        sdk.ListParameter,
				Value:       "lcov;cobertura;clover;other",
			},
			{
				Name:        "path",
				Description: `Path of the coverage report file.`,
				Type:        sdk.StringParameter,
			},
			{
				Name:        "minimum",
				Description: `Minimum percentage of coverage required (-1 means no minimum).`,
				Type:        sdk.NumberParameter,
				Advanced:    true,
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
						Format:  "cobertura",
						Minimum: "60",
						Path:    "./coverage.xml",
					},
				},
			},
		}},
	},
}
