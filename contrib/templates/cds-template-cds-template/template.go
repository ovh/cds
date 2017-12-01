package main

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/template"
)

type TemplateCDSTemplate struct {
	template.Common
}

func (t *TemplateCDSTemplate) Name() string {
	return "cds-template-cds-template"
}

func (t *TemplateCDSTemplate) Description() string {
	return `
This template creates a pipeline for building CDS Template with:

- A "Commit Stage" with one job "Compile"
- Job contains two steps: GitClone and CDS_GoBuild
`
}

func (t *TemplateCDSTemplate) Identifier() string {
	return "github.com/ovh/cds/contrib/templates/cds-template-cds-template/TemplateCDSTemplate"
}

func (t *TemplateCDSTemplate) Author() string {
	return "Yvonnick Esnault <yvonnick.esnault@corp.ovh.com>"
}

func (t *TemplateCDSTemplate) Type() string {
	return "BUILD"
}

func (t *TemplateCDSTemplate) Parameters() []sdk.TemplateParam {
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
			Value: "contrib/templates/{{.cds.application}}",
			Description: `Directory inside your repository where is the template.
Enter "contrib/templates/your-plugin" for github.com/ovh/cds/contrib/templates/your-plugin
			`,
		},
	}
}

func (t *TemplateCDSTemplate) ActionsNeeded() []string {
	return []string{
		"GitClone",
		"CDS_GoBuild",
	}
}

func (t *TemplateCDSTemplate) Apply(opts template.IApplyOptions) (sdk.Application, error) {
	a := sdk.Application{
		Name:       opts.ApplicationName(),
		ProjectKey: opts.ProjetKey(),
	}

	/* Build Pipeline */
	/* Build Pipeline - Commit Stage */
	jobCompile := sdk.Job{
		Action: sdk.Action{
			Name: "Compile CDS Template",
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
		{Name: "artifactUpload", Value: "true", Type: sdk.BooleanVariable},
	}

	/* Assemble Pipeline */
	a.Pipelines = []sdk.ApplicationPipeline{
		{
			Pipeline: sdk.Pipeline{
				Name:   "cds-template-build",
				Type:   sdk.BuildPipeline,
				Stages: []sdk.Stage{compileStage},
			},
		},
	}

	return a, nil
}

func main() {
	template.Main(&TemplateCDSTemplate{})
}
