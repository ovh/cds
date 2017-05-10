package main

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/template"
)

type TemplatePlain struct {
	template.Common
}

func (t *TemplatePlain) Name() string {
	return "cds-template-plain"
}

func (t *TemplatePlain) Description() string {
	return `
This template creates:

- a build pipeline with	two stages: Commit Stage and Packaging Stage
- a deploy pipeline with one stage: Deploy Stage

Commit Stage:

- run git clone
- run make build

Packaging Stage:

- run docker build and docker push

Deploy Stage:

- it's an empty script

Packaging and Deploy are optional.
`
}

func (t *TemplatePlain) Identifier() string {
	return "github.com/ovh/cds/contrib/templates/cds-template-plain/TemplatePlain"
}

func (t *TemplatePlain) Author() string {
	return "Yvonnick Esnault <yvonnick.esnault@corp.ovh.com>"
}

func (t *TemplatePlain) Type() string {
	return "BUILD"
}

func (t *TemplatePlain) Parameters() []sdk.TemplateParam {
	return []sdk.TemplateParam{
		{
			Name:        "repo",
			Type:        sdk.RepositoryVariable,
			Value:       "",
			Description: "Your source code repository",
		},
		{
			Name:        "withPackage",
			Type:        sdk.BooleanVariable,
			Value:       "withPackage",
			Description: "Do you want a Docker Package?",
		},
		{
			Name:        "withDeploy",
			Type:        sdk.BooleanVariable,
			Value:       "withDeploy",
			Description: "Do you want an deploy Pipeline?",
		},
	}
}

func (t *TemplatePlain) ActionsNeeded() []string {
	return []string{
		sdk.GitCloneAction,
	}
}

func (t *TemplatePlain) Apply(opts template.IApplyOptions) (sdk.Application, error) {
	//Return full application

	a := sdk.Application{
		Name:       opts.ApplicationName(),
		ProjectKey: opts.ProjetKey(),
	}

	/* Build Pipeline */
	/* Build Pipeline - Commit Stage */

	jobCompile := sdk.Job{
		Action: sdk.Action{
			Name: "Compile",
			Actions: []sdk.Action{
				sdk.Action{
					Name: sdk.GitCloneAction,
				},
				sdk.NewActionScript(`#!/bin/bash

set -xe

cd $(ls -1) && make`,

					[]sdk.Requirement{
						{
							Name:  "make",
							Type:  sdk.BinaryRequirement,
							Value: "make",
						},
					},
				),
			},
		},
	}

	compileStage := sdk.Stage{
		Name:       "Commit Stage",
		BuildOrder: 0,
		Enabled:    true,
		Jobs:       []sdk.Job{jobCompile},
	}

	/* Build Pipeline - Packaging Stage */

	jobDockerPackage := sdk.Job{
		Action: sdk.Action{
			Name: "Docker package",
			Actions: []sdk.Action{
				sdk.Action{
					Name: sdk.GitCloneAction,
				},
				sdk.NewActionScript(`#!/bin/bash
set -ex

cd $(ls -1)

docker build -t cds/{{.cds.application}}-{{.cds.version}} .
docker push cds/{{.cds.application}}-{{.cds.version}}`, []sdk.Requirement{
					{
						Name:  "bash",
						Type:  sdk.BinaryRequirement,
						Value: "bash",
					},
				},
				),
			},
		},
	}

	packagingStage := sdk.Stage{
		Name:       "Packaging Stage",
		BuildOrder: 0,
		Enabled:    true,
		Jobs:       []sdk.Job{jobDockerPackage},
	}

	/* Deploy Pipeline */
	/* Deploy Pipeline - Deploy Stage */

	jobDeploy := sdk.Job{
		Action: sdk.Action{
			Name: "Deploy",
			Actions: []sdk.Action{
				sdk.NewActionScript(`#!/bin/bash
set -ex

echo "CALL YOUR DEPLOY SCRIPT HERE"`, []sdk.Requirement{
					{
						Name:  "docker",
						Type:  sdk.BinaryRequirement,
						Value: "docker",
					},
				},
				),
			},
		},
	}

	deployStage := sdk.Stage{
		Name:       "Deploy Stage",
		BuildOrder: 0,
		Enabled:    true,
		Jobs:       []sdk.Job{jobDeploy},
	}

	/* Assemble Pipeline */

	a.Variable = []sdk.Variable{
		{Name: "repo", Value: opts.Parameters().Get("repo"), Type: sdk.StringVariable},
	}

	a.Pipelines = []sdk.ApplicationPipeline{
		{
			Pipeline: sdk.Pipeline{
				Name:   "build",
				Type:   sdk.BuildPipeline,
				Stages: []sdk.Stage{compileStage},
			},
			Triggers: []sdk.PipelineTrigger{
				{
					DestPipeline: sdk.Pipeline{
						Name: "deploy",
					},
					DestEnvironment: sdk.Environment{
						Name: "Production",
					},
				},
			},
		},
	}

	if opts.Parameters().Get("withPackage") == "true" {
		a.Pipelines[0].Pipeline.Stages = append(a.Pipelines[0].Pipeline.Stages, packagingStage)
	}

	if opts.Parameters().Get("withDeploy") == "true" {
		a.Pipelines = append(a.Pipelines,
			sdk.ApplicationPipeline{
				Pipeline: sdk.Pipeline{
					Name:   "deploy",
					Type:   sdk.DeploymentPipeline,
					Stages: []sdk.Stage{deployStage},
				},
			},
		)
	}

	return a, nil
}

func main() {
	template.Main(&TemplatePlain{})
}
