package action

import (
	"github.com/ovh/cds/sdk"
)

// JUnit action definition.
var JUnit = Manifest{
	Action: sdk.Action{
		Name: sdk.JUnitAction,
		Description: `CDS Builtin Action.
Parse given file to extract Unit Test results.`,
		Parameters: []sdk.Parameter{
			{
				Name:        "path",
				Description: `Path to junit xml file.`,
				Type:        sdk.TextParameter,
			},
		},
	},
}
