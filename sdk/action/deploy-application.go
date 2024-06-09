package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var deployExample = exportentities.StepDeploy("{{.cds.application}}")

// DeployApplication action definition.
var DeployApplication = Manifest{
	Action: sdk.Action{
		Name:        sdk.DeployApplicationAction,
		Description: `Deploy an application, useful only if you have a Deployment Platform associated to your current application.`,
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
					Deploy: &deployExample,
				},
			},
		}},
	},
}
