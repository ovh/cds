package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Promote action definition.
var Promote = Manifest{
	Action: sdk.Action{
		Name:        sdk.PromoteAction,
		Description: "This action promote artifacts in an artifact manager",
		Parameters: []sdk.Parameter{
			{
				Name:        "artifacts",
				Description: "(optional) Set a list of artifacts, separate by ','. You can also use regexp.",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "srcMaturity",
				Description: "Repository suffix from which the artifact will be moved",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "destMaturity",
				Description: "Repository suffix in which the artifact will be moved",
				Type:        sdk.StringParameter,
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
					Promote: &exportentities.StepPromote{
						Artifacts: "*.zip",
					},
				},
			},
		}},
	},
}
