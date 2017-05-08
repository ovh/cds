package main

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/template"
)

type TemplateCDSPlugin struct {
	template.Common
}

func (t *TemplateCDSPlugin) Name() string {
	return "cds-template-cds-plugin"
}

func (t *TemplateCDSPlugin) Description() string {
	return `
This template creates a pipeline for building CDS Plugin with:

- A "Commit Stage" with one job "Compile"
- Job contains two steps: GitClone and CDS_GoBuild
`
}

func (t *TemplateCDSPlugin) Identifier() string {
	return "github.com/ovh/cds/contrib/plugins/cds-template-cds-plugin/TemplateCDSPlugin"
}

func (t *TemplateCDSPlugin) Author() string {
	return "Yvonnick Esnault <yvonnick.esnault@corp.ovh.com>"
}

func (t *TemplateCDSPlugin) Type() string {
	return "BUILD"
}

func (t *TemplateCDSPlugin) Parameters() []sdk.TemplateParam {
	return []sdk.TemplateParam{
		{
			Name:        "repo",
			Type:        sdk.RepositoryVariable,
			Value:       "",
			Description: "Your source code repository",
		},
		{
			Name:        "package.root",
			Type:        sdk.StringVariable,
			Value:       "github.com/ovh/cds",
			Description: "example: github.com/ovh/cds",
		},
		{
			Name:  "package.sub",
			Type:  sdk.StringVariable,
			Value: "contrib/plugins/{{.cds.application}}",
			Description: `Directory inside your repository where is the plugin.
Enter "contrib/plugins/your-plugin" for github.com/ovh/cds/contrib/plugins/your-plugin
			`,
		},
	}
}

func (t *TemplateCDSPlugin) ActionsNeeded() []string {
	return []string{
		"GitClone",
		"CDS_GoBuild",
	}
}

func (t *TemplateCDSPlugin) Apply(opts template.IApplyOptions) (sdk.Application, error) {
	a := sdk.Application{
		Name:       opts.ApplicationName(),
		ProjectKey: opts.ProjetKey(),
	}

	/* Build Pipeline */
	/* Build Pipeline - Commit Stage */
	jobCompile := sdk.Job{
		Action: sdk.Action{
			Name: "Compile CDS Plugin",
			Actions: []sdk.Action{
				sdk.Action{
					Name: "GitClone",
					Parameters: []sdk.Parameter{
						{Name: "directory", Value: "./go/src/{{.cds.app.package.root}}"},
					},
				},
				sdk.Action{
					Name: "CDS_GoBuild",
					Parameters: []sdk.Parameter{
						{Name: "gopath", Value: "$HOME/go"}, // a gopath can't be relative "./go", so use $HOME/go
						{Name: "package", Value: "{{.cds.app.package.root}}/{{.cds.app.package.sub}}"},
						{Name: "binary", Value: "{{.cds.application}}"},
						{Name: "artifactUpload", Value: "true", Type: sdk.BooleanVariable},
					},
				},
			},
		},
	}

	compileStage := sdk.Stage{
		Name:       "Commit Stage",
		BuildOrder: 0,
		Enabled:    true,
		Jobs:       []sdk.Job{jobCompile},
	}

	a.Variable = []sdk.Variable{
		{Name: "repo", Value: opts.Parameters().Get("repo"), Type: sdk.StringVariable},
		{Name: "package.root", Value: opts.Parameters().Get("package.root"), Type: sdk.StringVariable},
		{Name: "package.sub", Value: opts.Parameters().Get("package.sub"), Type: sdk.StringVariable},
	}

	/* Assemble Pipeline */
	a.Pipelines = []sdk.ApplicationPipeline{
		{
			Pipeline: sdk.Pipeline{
				Name:   "cds-plugin-build",
				Type:   sdk.BuildPipeline,
				Stages: []sdk.Stage{compileStage},
			},
		},
	}

	return a, nil
}

func main() {
	template.Main(&TemplateCDSPlugin{})
}
