package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Release action definition.
var Release = Manifest{
	Action: sdk.Action{
		Name:        sdk.ReleaseAction,
		Description: "This action creates a release on a artifact manager. It promotes artifacts.",
		Parameters: []sdk.Parameter{
			{
				Name:        "releaseNote",
				Description: "(optional) Set a release note for the release.",
				Type:        sdk.TextParameter,
			},
			{
				Name:        "artifacts",
				Description: "(optional) Set a list of artifacts, separate by ','. You can also use regexp.",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "srcMaturity",
				Description: "Repository suffix from which the artifact will be moved",
				Type:        sdk.StringParameter,
				Value:       "snapshot",
				Advanced:    true,
			},
			{
				Name:        "destMaturity",
				Description: "Repository suffix in which the artifact will be moved",
				Type:        sdk.StringParameter,
				Value:       "release",
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
					Release: &exportentities.StepRelease{
						Artifacts: "(.*).zip",
					},
				},
			},
		}},
	},
}
