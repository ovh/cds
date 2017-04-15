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
This template creates a pipeline for building CDS Template.

Template contains:
- A "Commit Stage" with one job "Compile"
- Job contains two steps: CDS_GitClone and CDS_GoBuild
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
	}
}

func (t *TemplateCDSTemplate) ActionsNeeded() []string {
	return []string{
		"CDS_GitClone",
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
					Name: "CDS_GitClone",
					Parameters: []sdk.Parameter{
						{Name: "directory", Value: "./go/src/github.com/your-orga/your-repo"},
					},
				},
				sdk.Action{
					Name: "CDS_GoBuild",
					Parameters: []sdk.Parameter{
						{Name: "package", Value: "./go/src/github.com/your-orga/your-repo/templates/{{.cds.application}}"},
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

	/* Assemble Pipeline */
	a.Pipelines = []sdk.ApplicationPipeline{
		{
			Pipeline: sdk.Pipeline{
				Name:   "build",
				Type:   sdk.BuildPipeline,
				Stages: []sdk.Stage{compileStage},
			},
		},
	}

	return a, nil
}

func main() {
	p := TemplateCDSTemplate{}
	template.Serve(&p)
}
