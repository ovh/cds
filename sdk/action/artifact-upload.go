package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// ArtifactUpload action definition.
var ArtifactUpload = Manifest{
	Action: sdk.Action{
		Name:        sdk.ArtifactUpload,
		Description: "This action can be used to upload artifacts in CDS. This is the recommended way to share files between pipelines or stages.",
		Parameters: []sdk.Parameter{
			{
				Name:        "path",
				Type:        sdk.StringParameter,
				Description: "Path of file to upload, example: ./src/yourFile.json.",
			},
			{
				Name:        "tag",
				Type:        sdk.StringParameter,
				Description: "Artifact will be uploaded with a tag, generally {{.cds.version}}.",
				Value:       "{{.cds.version}}",
			},
			{
				Name:        "enabled",
				Type:        sdk.BooleanParameter,
				Description: "(optional) Enable artifact upload, \"true\" or \"false\".",
				Value:       "true",
				Advanced:    true,
			},
			{
				Name:        "destination",
				Description: "(optional) Destination of this artifact. Use the name of integration attached on your project.",
				Value:       "", // empty is the default value
				Type:        sdk.StringParameter,
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
					ArtifactUpload: &exportentities.StepArtifactUpload{
						Path: "{{.cds.workspace}}/myFile",
						Tag:  "{{.cds.version}}",
					},
				},
			},
		}},
	},
}
