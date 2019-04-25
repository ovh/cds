package action

import "github.com/ovh/cds/sdk"

// DeployApplication action definition.
var DeployApplication = Manifest{
	Action: sdk.Action{
		Name: sdk.DeployApplicationAction,
		Description: `CDS Builtin Action.
Deploy an application of a integration.`,
	},
}
