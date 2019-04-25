package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// ArtifactDownload action definition.
var ArtifactDownload = Manifest{
	Action: sdk.Action{
		Name:        sdk.ArtifactDownload,
		Description: "Allows you to download one or more artifacts in workspace.",
		Parameters: []sdk.Parameter{
			{
				Name:        "path",
				Description: "Path where artifacts will be downloaded",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "tag",
				Description: "Artifact are uploaded with a tag, generally {{.cds.version}}",
				Type:        sdk.StringParameter,
				Value:       "{{.cds.version}}",
			},
			{
				Name:        "enabled",
				Type:        sdk.BooleanParameter,
				Description: "Enable artifact download",
				Value:       "true",
				Advanced:    true,
			},
			{
				Name:        "pattern",
				Type:        sdk.StringParameter,
				Description: "Empty: download all files. Otherwise, enter regexp pattern to choose file: (fileA|fileB)",
				Value:       "",
				Advanced:    true,
			},
		},
	},
	Example: exportentities.Step{
		ArtifactDownload: &exportentities.StepArtifactDownload{
			Path:    "{{.cds.workspace}}",
			Pattern: "*.tag.gz",
			Tag:     "{{.cds.version}}",
		},
	},
}
