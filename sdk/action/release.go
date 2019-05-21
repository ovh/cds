package action

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Release action definition.
var Release = Manifest{
	Action: sdk.Action{
		Name:        sdk.ReleaseAction,
		Description: "This action creates a release on the git repository linked to the application, if repository manager implements it.",
		Parameters: []sdk.Parameter{
			{
				Name:        "tag",
				Description: "Tag attached to the release.",
				Value:       "{{.cds.release.version}}",
				Type:        sdk.StringParameter,
			},
			{
				Name:        "title",
				Value:       "",
				Description: "Set a title for the release.",
				Type:        sdk.StringParameter,
			},
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
		},
	},
	Example: exportentities.PipelineV1{
		Version: exportentities.PipelineVersion1,
		Name:    "Pipeline1",
		Parameters: map[string]exportentities.ParameterValue{
			"tagLevel": exportentities.ParameterValue{
				Type:         "list",
				DefaultValue: "major;minor;patch",
				Description:  "major, minor or patch",
			},
		},
		Stages: []string{"Stage1"},
		Jobs: []exportentities.Job{{
			Name:  "Job1",
			Stage: "Stage1",
			Steps: []exportentities.Step{
				{
					Checkout: &checkoutExample,
				},
				{
					GitTag: &exportentities.StepGitTag{
						Path:       "{{.cds.workspace}}",
						TagLevel:   "{{.cds.pip.tagLevel}}",
						TagMessage: "Release from CDS run {{.cds.version}}",
					},
				},
				{
					Script: []string{
						"#!/bin/sh",
						"TAG=`git describe --abbrev=0 --tags`",
						"worker export tag $TAG",
					},
				},
				{
					Release: &exportentities.StepRelease{
						Artifacts:   "{{.cds.workspace}}/myFile",
						Title:       "{{.cds.build.tag}}",
						ReleaseNote: "My release {{.cds.build.tag}}",
						Tag:         "{{.cds.build.tag}}",
					},
				},
			},
		}},
	},
}
