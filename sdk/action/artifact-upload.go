package action

import "github.com/ovh/cds/sdk"

// ArtifactUpload action definition.
var ArtifactUpload = Manifest{
	Action: sdk.Action{
		Name:        sdk.ArtifactUpload,
		Description: "Allows you to upload one or more artifacts from workspace.",
		Parameters: []sdk.Parameter{
			{
				Name:        "path",
				Type:        sdk.StringParameter,
				Description: "Path of file to upload, example: ./src/yourFile.json",
			},
			{
				Name:        "tag",
				Type:        sdk.StringParameter,
				Description: "Artifact will be uploaded with a tag, generally {{.cds.version}}",
				Value:       "{{.cds.version}}",
			},
			{
				Name:        "enabled",
				Type:        sdk.BooleanParameter,
				Description: "Enable artifact upload",
				Value:       "true",
				Advanced:    true,
			},
			{
				Name:        "destination",
				Description: "Destination of this artifact. Use the name of integration attached on your project",
				Value:       "", // empty is the default value
				Type:        sdk.StringParameter,
				Advanced:    true,
			},
		},
	},
}
