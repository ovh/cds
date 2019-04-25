package action

import (
	"github.com/ovh/cds/sdk"
)

// CheckoutApplication action definition.
var CheckoutApplication = Manifest{
	Action: sdk.Action{
		Name: sdk.CheckoutApplicationAction,
		Description: `CDS Builtin Action.
Checkout a repository into a new directory.

This action use the configuration from application to git clone the repository.
The clone will be done with a depth of 50 and with submodules.
If you want to modify theses options, you have to use gitClone action.
`,
		Parameters: []sdk.Parameter{
			{
				Name:        "directory",
				Description: "The name of a directory to clone into.",
				Value:       "{{.cds.workspace}}",
				Type:        sdk.StringParameter,
			},
		},
		Requirements: []sdk.Requirement{
			{
				Name:  "git",
				Type:  sdk.BinaryRequirement,
				Value: "git",
			},
		},
	},
}
