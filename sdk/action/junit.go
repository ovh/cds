package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var exampleJUnit = exportentities.StepJUnitReport("{{.cds.workspace}}/report.xml")

// JUnit action definition.
var JUnit = Manifest{
	Action: sdk.Action{
		Name:        sdk.JUnitAction,
		Description: "This action parses a given Junit formatted XML file to extract its test results.",
		Parameters: []sdk.Parameter{
			{
				Name:        "path",
				Description: `Path to junit xml file.`,
				Type:        sdk.TextParameter,
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
					JUnitReport: &exampleJUnit,
				},
			},
		}},
	},
}
