package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var installKeyExample = exportentities.StepInstallKey("proj-mykey")

// InstallKey action definition.
var InstallKey = Manifest{
	Action: sdk.Action{
		Name: sdk.InstallKeyAction,
		Description: `CDS Builtin Action.
Checkout a repository into a new directory.

This action installs a SSH/PGP key generated in CDS. And if it's a SSH key it will export in the environment variable named PKEY the path to the private key.
For example to use with 'ssh -i $PKEY'
`,
		Parameters: []sdk.Parameter{
			{
				Name:        "key",
				Value:       "",
				Description: `Set the key to install in your workspace`,
				Type:        sdk.KeyParameter,
			},
			{
				Name:        "file",
				Value:       "",
				Description: `Write key to destination file`,
				Type:        sdk.StringParameter,
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
					InstallKey: &installKeyExample,
				},
			},
		}},
	},
}
