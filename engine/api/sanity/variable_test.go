package sanity

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func Test_loadUsedVariables(t *testing.T) {
	log.SetLogger(t)

	tests := []struct {
		name            string
		a               *sdk.Action
		projectVars     []string
		applicationVars []string
		envVars         []string
		gitVars         []string
		badVars         []string
	}{
		{
			name: "Check project, application, env and git variables",
			a: &sdk.Action{
				Parameters: []sdk.Parameter{
					{
						Value: "{{.cds.proj.proj_var}}",
					}, {
						Value: "{{.cds.app.app_var}}",
					}, {
						Value: "{{.cds.env.env_var}}",
					}, {
						Value: "{{.git.git_var}}",
					},
				},
			},
			projectVars:     []string{"proj_var"},
			applicationVars: []string{"app_var"},
			envVars:         []string{"env_var"},
			gitVars:         []string{"git_var"},
		},
		{
			name: "Check project, application, env and git variables recursively",
			a: &sdk.Action{
				Parameters: []sdk.Parameter{
					{
						Value: "{{.cds.proj.proj_var}}",
					}, {
						Value: "{{.cds.app.app_var}}",
					}, {
						Value: "{{.cds.env.env_var}}",
					}, {
						Value: "{{.git.git_var}}",
					},
				},
				Actions: []sdk.Action{
					{
						Parameters: []sdk.Parameter{
							{
								Value: "{{.cds.proj.proj_var1}}",
							}, {
								Value: "{{.cds.app.app_var1}}",
							}, {
								Value: "{{.cds.env.env_var1}}",
							}, {
								Value: "{{.git.git_var1}}",
							},
						},
						Actions: []sdk.Action{
							{
								Parameters: []sdk.Parameter{
									{
										Value: "{{.cds.proj.proj_var2}}",
									}, {
										Value: "{{.cds.app.app_var2}}",
									}, {
										Value: "{{.cds.env.env_var2}}",
									}, {
										Value: "{{.git.git_var2}}",
									},
								},
							},
						},
					},
				},
			},
			projectVars:     []string{"proj_var", "proj_var1", "proj_var2"},
			applicationVars: []string{"app_var", "app_var1", "app_var2"},
			envVars:         []string{"env_var", "env_var1", "env_var2"},
			gitVars:         []string{"git_var", "git_var1", "git_var2"},
		}, {
			name: "Check bad project, application, env and git variables",
			a: &sdk.Action{
				Parameters: []sdk.Parameter{
					{
						Value: "{{ .cds.proj.proj_var }}",
					}, {
						Value: "{{cds.app.app_var}}",
					}, {
						Value: "{{ .cds.env}}",
					}, {
						Value: "{{ .git.git_var}}",
					},
				},
			},
			projectVars:     []string{},
			applicationVars: []string{},
			envVars:         []string{},
			gitVars:         []string{},
			badVars:         []string{"{{ .cds.proj.proj_var }}", "{{cds.app.app_var}}", "{{ .cds.env}}", "{{ .git.git_var}}"},
		},
	}
	for _, tt := range tests {
		projectVars, applicationVars, envVars, gitVars, badVars := loadUsedVariables(tt.a)

		test.EqualValuesWithoutOrder(t, projectVars, tt.projectVars, "%q. loadUsedVariables() got = %v, want %v", tt.name, projectVars, tt.projectVars)

		test.EqualValuesWithoutOrder(t, applicationVars, tt.applicationVars, "%q. loadUsedVariables() got = %v, want %v", tt.name, applicationVars, tt.applicationVars)

		test.EqualValuesWithoutOrder(t, envVars, tt.envVars, "%q. loadUsedVariables() got = %v, want %v", tt.name, envVars, tt.envVars)

		test.EqualValuesWithoutOrder(t, gitVars, tt.gitVars, "%q. loadUsedVariables() got = %v, want %v", tt.name, gitVars, tt.gitVars)

		test.EqualValuesWithoutOrder(t, badVars, tt.badVars, "%q. loadUsedVariables() got = %v, want %v", tt.name, badVars, tt.badVars)

	}
}
