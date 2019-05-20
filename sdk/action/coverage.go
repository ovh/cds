package action

import "github.com/ovh/cds/sdk"

// Coverage action definition.
var Coverage = Manifest{
	Action: sdk.Action{
		Name: sdk.CoverageAction,
		Description: `CDS Builtin Action.
Parse given file to extract coverage results.`,
		Parameters: []sdk.Parameter{
			{
				Name:        "format",
				Description: `Coverage report format.`,
				Type:        sdk.ListParameter,
				Value:       "lcov;cobertura",
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
}
