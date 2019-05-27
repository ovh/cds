package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var checkoutExample = exportentities.StepCheckout("{{.cds.workspace}}")

// CheckoutApplication action definition.
var CheckoutApplication = Manifest{
	Action: sdk.Action{
		Name: sdk.CheckoutApplicationAction,
		Description: `CDS Builtin Action.
Checkout a repository into a new directory.

This action use the configuration from application vcs strategy to git clone the repository.
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
	Example: exportentities.PipelineV1{
		Version: exportentities.PipelineVersion1,
		Name:    "Pipeline1",
		Stages:  []string{"Stage1"},
		Jobs: []exportentities.Job{{
			Name:  "Job1",
			Stage: "Stage1",
			Steps: []exportentities.Step{
				{
					Checkout: &checkoutExample,
				},
			},
		}},
	},
}
