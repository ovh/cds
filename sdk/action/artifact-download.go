package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// ArtifactDownload action definition.
var ArtifactDownload = Manifest{
	Action: sdk.Action{
		Name:        sdk.ArtifactDownload,
		Description: "This action can be used to retrieve an artifact previously uploaded by an Artifact Upload.",
		Parameters: []sdk.Parameter{
			{
				Name:        "path",
				Description: "Path where artifacts will be downloaded.",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "tag",
				Description: "Artifact are uploaded with a tag, generally {{.cds.version}}.",
				Type:        sdk.StringParameter,
				Value:       "{{.cds.version}}",
			},
			{
				Name:        "enabled",
				Type:        sdk.BooleanParameter,
				Description: "(optional) Enable artifact download.",
				Value:       "true",
				Advanced:    true,
			},
			{
				Name:        "pattern",
				Type:        sdk.StringParameter,
				Description: "(optional) Empty: download all files. Otherwise, enter regexp pattern to choose file: (fileA|fileB).",
				Value:       "",
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
					ArtifactDownload: &exportentities.StepArtifactDownload{
						Path:    "{{.cds.workspace}}",
						Pattern: "*.tag.gz",
						Tag:     "{{.cds.version}}",
					},
				},
			},
		}},
	},
}
