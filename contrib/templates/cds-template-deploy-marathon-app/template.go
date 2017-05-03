package main

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/template"
)

// TemplateMarathonApp for deploying a marathon application
type TemplateMarathonApp struct {
	template.Common
}

// Name of template, should be exactly same name as binary
func (t *TemplateMarathonApp) Name() string {
	return "cds-template-deploy-marathon-app"
}

// Description of this template
func (t *TemplateMarathonApp) Description() string {
	return `
This template creates:

- a deployment pipeline with one stage, and containing one job
- job calls plugin-marathon
- an application with a variable named "marathon.config"
- uses environment variables marathonHost, password and user

Please update Application / Environment Variables after creating application.
`
}

// Identifier of this template
func (t *TemplateMarathonApp) Identifier() string {
	return "github.com/ovh/cds/contrib/templates/cds-template-deploy-marathon-app/TemplateMarathonApp"
}

// Author of this template
func (t *TemplateMarathonApp) Author() string {
	return "Yvonnick Esnault <yvonnick.esnault@corp.ovh.com>"
}

// Type of template
func (t *TemplateMarathonApp) Type() string {
	return "BUILD"
}

// Parameters contains template parameters
func (t *TemplateMarathonApp) Parameters() []sdk.TemplateParam {
	return []sdk.TemplateParam{
		{
			Name:        "docker.image",
			Type:        sdk.StringVariable,
			Value:       "<your-docker-registry>/<your-namespace>/<your-app>",
			Description: "Your docker image without the tag",
		}, {
			Name:        "marathon.appID",
			Type:        sdk.StringVariable,
			Value:       "/{{.cds.environment}}/<your-app>",
			Description: "Your marathon application ID",
		}, {
			Name: "marathon.config",
			Type: sdk.TextVariable,
			Value: `{
    "id": "{{.cds.app.marathon.appID}}",
    "mem": 256,
    "cpus": 0.1,
    "instances": 1,
    "container": {
        "type": "DOCKER",
        "docker": {
            "network": "BRIDGE",
            "portMappings": [
                {
                    "protocol": "tcp",
                    "containerPort": 8080,
                    "hostPort": 0
                }
            ],
            "image": "{{.cds.app.docker.image}}:{{.tag}}",
            "forcePullImage": false
        }
    },
    "env": {
        "EXAMPLE_ENV_1" : "EXAMPLE_VALUE_1"
    },
    "labels": {
        "LB_0_MODE": "http",
        "LB_0_VHOST": "{{.cds.app.marathon.vHost}}"
    },
    "healthChecks": [
        {
            "path": "/mon/ping",
            "protocol": "HTTP",
            "portIndex": 0,
            "gracePeriodSeconds": 15,
            "intervalSeconds": 60,
            "timeoutSeconds": 10,
            "maxConsecutiveFailures": 2,
            "ignoreHttp1xx": false
        }
    ],
    "constraints": [["hostname", "GROUP_BY"]]
}`,
			Description: "Content of your marathon.json file",
		},
	}
}

// ActionsNeeded contains action needed by this template
func (t *TemplateMarathonApp) ActionsNeeded() []string {
	return []string{
		"plugin-marathon",
	}
}

// Apply returns full application
func (t *TemplateMarathonApp) Apply(opts template.IApplyOptions) (sdk.Application, error) {

	job := sdk.Job{
		Action: sdk.Action{
			Name:        "Marathon Deploy",
			Description: "Deploy application on marathon",
			Actions: []sdk.Action{
				sdk.NewActionScript(`
#!/bin/bash
set -e
cat << EOF > marathon.{{.cds.application}}.json
{{.cds.app.marathon.config}}
EOF`, nil),
				sdk.Action{
					Name: "plugin-marathon",
					Parameters: []sdk.Parameter{
						{Name: "configuration", Value: "marathon.{{.cds.application}}.json"},
						{Name: "waitForDeployment", Value: "true"},
						{Name: "insecureSkipVerify", Value: "true"},
					},
				},
			},
		},
	}

	/* Assemble Pipeline */
	a := sdk.Application{
		Name:       opts.ApplicationName(),
		ProjectKey: opts.ProjetKey(),
	}

	// cds API will write repo value
	a.Variable = []sdk.Variable{
		{Name: "docker.image", Value: opts.Parameters().Get("docker.image"), Type: sdk.StringVariable},
		{Name: "marathon.appID", Value: opts.Parameters().Get("marathon.appID"), Type: sdk.StringVariable},
		{Name: "marathon.config", Value: opts.Parameters().Get("marathon.config"), Type: sdk.TextVariable},
		{Name: "marathon.vHost", Value: "{{.cds.application}}.{{.cds.env.marathon.lb}}", Type: sdk.StringVariable},
	}

	stage := sdk.Stage{
		Name:       "Deployement Stage",
		BuildOrder: 1,
		Enabled:    true,
		Jobs:       []sdk.Job{job},
	}

	a.Pipelines = []sdk.ApplicationPipeline{
		{
			Pipeline: sdk.Pipeline{
				Name:   "Deploy",
				Type:   sdk.DeploymentPipeline,
				Stages: []sdk.Stage{stage},
				Parameter: []sdk.Parameter{
					{
						Name:        "tag",
						Type:        sdk.StringParameter,
						Description: "Docker image tag",
					},
				},
			},
		},
	}

	return a, nil
}

func main() {
	template.Main(&TemplateMarathonApp{})
}
