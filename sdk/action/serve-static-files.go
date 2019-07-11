package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// ServeStaticFiles action definition.
var ServeStaticFiles = Manifest{
	Action: sdk.Action{
		Name:        sdk.ServeStaticFiles,
		Enabled:     false,
		Description: "This action can be used to upload static files and serve them. For example your HTML report about coverage, tests, performances, ...",
		Parameters: []sdk.Parameter{
			{
				Name:        "name",
				Description: "Name to display in CDS UI and identify your static files.",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "path",
				Description: "Path where static files will be uploaded (example: mywebsite/*). If it's a file, the entrypoint would be set to this filename by default.",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "entrypoint",
				Description: "(optional) Filename (and not path) for the entrypoint when serving static files (default: if empty it would be index.html).",
				Type:        sdk.StringParameter,
				Value:       "",
				Advanced:    true,
			},
			{
				Name:        "static-key",
				Description: "(optional) Indicate a static-key which will be a reference to keep the same generated URL. Example: {{.git.branch}}.",
				Type:        sdk.StringParameter,
				Value:       "",
				Advanced:    true,
			},
			{
				Name:        "destination",
				Description: "(optional) Destination of uploading. Use the name of integration attached on your project.",
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
					ServeStaticFiles: &exportentities.StepServeStaticFiles{
						Name: "mywebsite",
						Path: "mywebsite/*",
					},
				},
			},
		}},
	},
}
