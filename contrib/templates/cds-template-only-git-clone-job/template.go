package main

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/template"
)

type TemplateOnlyGitCloneJob struct {
	template.Common
}

func (t *TemplateOnlyGitCloneJob) Name() string {
	return "cds-template-only-git-clone-job"
}

func (t *TemplateOnlyGitCloneJob) Description() string {
	return `
This template creates:

- a build pipeline with	one stage, containing one job
- job contains 2 steps: GitClone and a empty script.

Pipeline name contains Application name.
If you want to make a reusable pipeline, please consider updating this name after creating application.
`
}

func (t *TemplateOnlyGitCloneJob) Identifier() string {
	return "github.com/ovh/cds/contrib/templates/cds-template-only-git-clone/TemplateOnlyGitCloneJob"
}

func (t *TemplateOnlyGitCloneJob) Author() string {
	return "Yvonnick Esnault <yvonnick.esnault@corp.ovh.com>"
}

func (t *TemplateOnlyGitCloneJob) Type() string {
	return "BUILD"
}

func (t *TemplateOnlyGitCloneJob) Parameters() []sdk.TemplateParam {
	return []sdk.TemplateParam{
		{
			Name:        "repo",
			Type:        sdk.RepositoryVariable,
			Value:       "",
			Description: "Your source code repository",
		},
	}
}

func (t *TemplateOnlyGitCloneJob) ActionsNeeded() []string {
	return []string{
		sdk.GitCloneAction,
	}
}

func (t *TemplateOnlyGitCloneJob) Apply(opts template.IApplyOptions) (sdk.Application, error) {
	a := sdk.Application{
		Name:       opts.ApplicationName(),
		ProjectKey: opts.ProjetKey(),
	}

	/* Build Pipeline - Stage */
	jobCompile := sdk.Job{
		Action: sdk.Action{
			Name: "Compile",
			Actions: []sdk.Action{
				sdk.Action{
					Name: sdk.GitCloneAction,
				},
				sdk.NewActionScript(`#!/bin/bash

set -xe

echo "TODO"`,
					[]sdk.Requirement{
						{
							Name:  "echo",
							Type:  sdk.BinaryRequirement,
							Value: "echo",
						},
					},
				),
			},
		},
	}

	compileStage := sdk.Stage{
		Name:       "First Stage",
		BuildOrder: 0,
		Enabled:    true,
		Jobs:       []sdk.Job{jobCompile},
	}

	/* Assemble Pipeline */
	a.Variable = []sdk.Variable{
		{Name: "repo", Value: opts.Parameters().Get("repo"), Type: sdk.StringVariable},
	}

	a.Pipelines = []sdk.ApplicationPipeline{
		{
			Pipeline: sdk.Pipeline{
				Name:   opts.ApplicationName() + "-build",
				Type:   sdk.BuildPipeline,
				Stages: []sdk.Stage{compileStage},
			},
		},
	}

	return a, nil
}

func main() {
	template.Main(&TemplateOnlyGitCloneJob{})
}
