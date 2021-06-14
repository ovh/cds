package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var pushBuildInfoExample = exportentities.StepPushBuildInfo("{{.cds.workflow}}")

// PushBuildInfo action definition.
var PushBuildInfo = Manifest{
	Action: sdk.Action{
		Name:        sdk.PushBuildInfo,
		Description: `Push build info into an artifact manager, useful only if you have an artifact manager linked to your workflow.`,
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
					PushBuildInfo: &pushBuildInfoExample,
				},
			},
		}},
	},
}
