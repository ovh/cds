package action

import (
	"github.com/ovh/cds/sdk"
)

// Release action definition.
var Release = Manifest{
	Action: sdk.Action{
		Name:        sdk.ReleaseAction,
		Description: `CDS Builtin Action. Make a release using repository manager.`,
		Parameters: []sdk.Parameter{
			{
				Name:        "tag",
				Description: "Tag name.",
				Value:       "{{.cds.release.version}}",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "title",
				Value:       "",
				Description: "Set a title for the release",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "releaseNote",
				Description: "Set a release note for the release",
				Type:        sdk.TextParameter,
			},
			{
				Name:        "artifacts",
				Description: "Set a list of artifacts, separate by , . You can also use regexp.",
				Type:        sdk.StringParameter,
			},
		},
	},
}
