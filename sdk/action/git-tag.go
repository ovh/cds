package action

import (
	"github.com/ovh/cds/sdk"
)

// GitTag action definition.
var GitTag = Manifest{
	Action: sdk.Action{
		Name: sdk.GitTagAction,
		Description: `CDS Builtin Action.
Tag the current branch and push it.
Semver used if fully compatible with https://semver.org/
`,
		Parameters: []sdk.Parameter{
			{
				Name:        "tagPrerelease",
				Description: "Prerelease version of the tag. Example: alpha on a tag 1.0.0 will return 1.0.0-apha",
				Value:       "",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "tagLevel",
				Description: "Set the level of the tag. Must be 'major' or 'minor' or 'patch'",
				Value:       "",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "tagMetadata",
				Description: "Metadata of the tag. Example: cds.42 on a tag 1.0.0 will return 1.0.0+cds.42",
				Value:       "",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "tagMessage",
				Description: "Set a message for the tag.",
				Value:       "",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "path",
				Description: "The path to your git directory.",
				Value:       "{{.cds.workspace}}",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "prefix",
				Description: "Prefix for tag name",
				Value:       "",
				Type:        sdk.StringParameter,
				Advanced:    true,
			},
		},
		Requirements: []sdk.Requirement{
			{
				Name:  "git",
				Type:  sdk.BinaryRequirement,
				Value: "git",
			},
			{
				Name:  "gpg",
				Type:  sdk.BinaryRequirement,
				Value: "gpg",
			},
		},
	},
}
