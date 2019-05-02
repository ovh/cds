package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// GitTag action definition.
var GitTag = Manifest{
	Action: sdk.Action{
		Name: sdk.GitTagAction,
		Description: `Tag the current branch and push it. Use vcs config from your application.
Semver used if fully compatible with https://semver.org.
`,
		Parameters: []sdk.Parameter{
			{
				Name:        "tagLevel",
				Description: "Set the level of the tag. Must be 'major' or 'minor' or 'patch'.",
				Value:       "",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "tagPrerelease",
				Description: "(optional) Prerelease version of the tag. Example: alpha on a tag 1.0.0 will return 1.0.0-alpha.",
				Value:       "",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "tagMetadata",
				Description: "(optional) Metadata of the tag. Example: cds.42 on a tag 1.0.0 will return 1.0.0+cds.42.",
				Value:       "",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "tagMessage",
				Description: "(optional) Set a message for the tag.",
				Value:       "",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "path",
				Description: "(optional) The path to your git directory.",
				Value:       "{{.cds.workspace}}",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "prefix",
				Description: "(optional) Add a prefix for tag name.",
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
	Example: exportentities.PipelineV1{
		Version: exportentities.PipelineVersion1,
		Name:    "Pipeline1",
		Parameters: map[string]exportentities.ParameterValue{
			"tagLevel": exportentities.ParameterValue{
				Type:         "list",
				DefaultValue: "major;minor;patch",
				Description:  "major, minor or patch",
			},
		},
		Stages: []string{"Stage1"},
		Jobs: []exportentities.Job{{
			Name:  "Job1",
			Stage: "Stage1",
			Steps: []exportentities.Step{
				{
					Checkout: &checkoutExample,
				},
				{
					GitTag: &exportentities.StepGitTag{
						Path:       "{{.cds.workspace}}",
						TagLevel:   "{{.cds.pip.tagLevel}}",
						TagMessage: "Release from CDS run {{.cds.version}}",
					},
				},
			},
		}},
	},
}
