package exportentities

import (
	"text/template"

	"github.com/ovh/cds/sdk"
)

// Application represents exported sdk.Application
type Application struct {
	Name              string                         `json:"name" yaml:"name"`
	RepositoryManager string                         `json:"repo_manager,omitempty" yaml:"repo_manager,omitempty"`
	RepositoryName    string                         `json:"repo_name,omitempty" yaml:"repo_name,omitempty"`
	Permissions       map[string]int                 `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Variables         map[string]VariableValue       `json:"variables,omitempty" yaml:"variables,omitempty"`
	Pipelines         map[string]ApplicationPipeline `json:"pipelines,omitempty" yaml:"pipelines,omitempty"`
}

// ApplicationPipeline represents exported sdk.ApplicationPipeline
type ApplicationPipeline struct {
	Parameters map[string]VariableValue              `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Triggers   map[string]ApplicationPipelineTrigger `json:"triggers,omitempty" yaml:"triggers,omitempty"`
	Options    []ApplicationPipelineOptions          `json:"options,omitempty" yaml:"options,omitempty"`
}

// ApplicationPipelineOptions represents presence of hooks, pollers, notifications and scheduler for an tuple application pipeline environment
type ApplicationPipelineOptions struct {
	Environment   *string                                    `json:"environment,omitempty" yaml:"environment,omitempty"`
	Hook          *bool                                      `json:"hook,omitempty" yaml:"hook,omitempty"`
	Polling       *bool                                      `json:"polling,omitempty" yaml:"polling,omitempty"`
	Notifications map[string]ApplicationPipelineNotification `json:"notifications,omitempty" yaml:"notifications,omitempty"`
	Schedulers    []ApplicationPipelineScheduler             `json:"schedulers,omitempty" yaml:"schedulers,omitempty"`
}

// ApplicationPipelineScheduler represents exported sdk.PipelineScheduler
type ApplicationPipelineScheduler struct {
	CronExpr   string                   `json:"cron_expr" yaml:"cron_expr"`
	Parameters map[string]VariableValue `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// ApplicationPipelineNotification represents exported notification
type ApplicationPipelineNotification sdk.UserNotificationSettings

// ApplicationPipelineTrigger represents an exported pipeline trigger
type ApplicationPipelineTrigger struct {
	ProjectKey      *string     `json:"project_key" yaml:"project_key"`
	ApplicationName *string     `json:"application_name" yaml:"application_name"`
	FromEnvironment *string     `json:"from_environment,omitempty" yaml:"from_environment,omitempty"`
	ToEnvironment   *string     `json:"to_environment,omitempty" yaml:"to_environment,omitempty"`
	Manual          bool        `json:"manual" yaml:"manual"`
	Conditions      []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// Condition represents sdk.Prerequisite
type Condition struct {
	Variable string `json:"variable" yaml:"variable"`
	Expected string `json:"expected" yaml:"expected"`
}

// NewApplication instanciance an exportable application from an sdk.Application
func NewApplication(app *sdk.Application) (a *Application) {
	a = new(Application)
	a.Name = app.Name

	if app.RepositoriesManager != nil {
		a.RepositoryManager = app.RepositoriesManager.Name
		a.RepositoryName = app.RepositoryFullname
	}

	a.Variables = make(map[string]VariableValue, len(app.Variable))
	for _, v := range app.Variable {
		a.Variables[v.Name] = VariableValue{
			Type:  string(v.Type),
			Value: v.Value,
		}
	}
	a.Permissions = make(map[string]int, len(app.ApplicationGroups))
	for _, p := range app.ApplicationGroups {
		a.Permissions[p.Group.Name] = p.Permission
	}

	a.Pipelines = make(map[string]ApplicationPipeline, len(app.Pipelines))
	for _, ap := range app.Pipelines {
		pip := ApplicationPipeline{}

		pip.Parameters = make(map[string]VariableValue, len(ap.Parameters))
		for _, param := range ap.Parameters {
			pip.Parameters[param.Name] = VariableValue{
				Type:  string(param.Type),
				Value: param.Value,
			}
		}

		pip.Triggers = make(map[string]ApplicationPipelineTrigger, len(ap.Triggers))
		for _, t := range ap.Triggers {

			c := make([]Condition, len(t.Prerequisites))
			var i int
			for _, pr := range t.Prerequisites {
				c[i] = Condition{
					Variable: pr.Parameter,
					Expected: pr.ExpectedValue,
				}
			}

			var srcEnv, destEnv, pKey, appName *string
			if t.SrcEnvironment.Name != sdk.DefaultEnv.Name {
				srcEnv = &t.SrcEnvironment.Name
			}
			if t.DestEnvironment.Name != sdk.DefaultEnv.Name {
				destEnv = &t.DestEnvironment.Name
			}
			if t.DestProject.Key != app.ProjectKey {
				pKey = &t.DestProject.Key
			}
			if t.DestApplication.Name != a.Name {
				appName = &t.DestApplication.Name
			}
			pip.Triggers[t.DestPipeline.Name] = ApplicationPipelineTrigger{
				ProjectKey:      pKey,
				ApplicationName: appName,
				ToEnvironment:   destEnv,
				FromEnvironment: srcEnv,
				Manual:          t.Manual,
				Conditions:      c,
			}
		}

		mapEnvOpts := map[string]*ApplicationPipelineOptions{}
		//Hooks
		for _, h := range app.Hooks {
			if h.Enabled && h.Pipeline.Name == ap.Pipeline.Name {
				if _, ok := mapEnvOpts[sdk.DefaultEnv.Name]; !ok {
					mapEnvOpts[sdk.DefaultEnv.Name] = &ApplicationPipelineOptions{}
				}
				o := mapEnvOpts[sdk.DefaultEnv.Name]
				if h.Enabled {
					var ok = true
					o.Hook = &ok
				}
			}
		}

		//Pollers
		for _, p := range app.RepositoryPollers {
			if p.Enabled && p.Pipeline.Name == ap.Pipeline.Name {
				if _, ok := mapEnvOpts[sdk.DefaultEnv.Name]; !ok {
					mapEnvOpts[sdk.DefaultEnv.Name] = &ApplicationPipelineOptions{}
				}
				o := mapEnvOpts[sdk.DefaultEnv.Name]
				var ok = true
				o.Polling = &ok
			}

		}

		//Notifications
		for _, n := range app.Notifications {
			if ap.Pipeline.Name == n.Pipeline.Name {
				if _, ok := mapEnvOpts[n.Environment.Name]; !ok {
					mapEnvOpts[n.Environment.Name] = &ApplicationPipelineOptions{}
				}
				o := mapEnvOpts[n.Environment.Name]
				for t, n := range n.Notifications {
					if o.Notifications == nil {
						o.Notifications = make(map[string]ApplicationPipelineNotification)
					}
					o.Notifications[string(t)] = n
				}
			}
		}

		//Schedulers
		for _, s := range app.Schedulers {
			if ap.Pipeline.ID == s.PipelineID {
				if _, ok := mapEnvOpts[s.EnvironmentName]; !ok {
					mapEnvOpts[s.EnvironmentName] = &ApplicationPipelineOptions{}
				}
				o := mapEnvOpts[s.EnvironmentName]
				if o.Schedulers == nil {
					o.Schedulers = []ApplicationPipelineScheduler{}
				}
				aps := ApplicationPipelineScheduler{
					CronExpr: s.Crontab,
				}
				aps.Parameters = make(map[string]VariableValue, len(s.Args))
				for _, p := range s.Args {
					aps.Parameters[p.Name] = VariableValue{Type: string(p.Type), Value: p.Value}
				}
				o.Schedulers = append(o.Schedulers, aps)
			}
		}

		//Compute all
		pip.Options = make([]ApplicationPipelineOptions, len(mapEnvOpts))
		var i int
		for k, v := range mapEnvOpts {
			if k != sdk.DefaultEnv.Name {
				s := k
				pip.Options[i].Environment = &s
			}
			if v.Hook != nil {
				pip.Options[i].Hook = v.Hook
			}
			if v.Polling != nil {
				pip.Options[i].Polling = v.Polling
			}
			pip.Options[i].Notifications = v.Notifications
			pip.Options[i].Schedulers = v.Schedulers

			i++
		}

		a.Pipelines[ap.Pipeline.Name] = pip
	}

	return
}

//HCLTemplate returns text/template
func (a *Application) HCLTemplate() (*template.Template, error) {
	tmpl := `name = "{{.Name}}"

repo_manager = "{{.RepositoryManager}}
repo_name = "{{.RepositoryName}}

permissions = { {{ range $key, $value := .Permissions }}
	"{{$key}}" = {{$value}}{{ end }}
}

variables = { 
{{ range $key, $value := .Variables }}
	"{{ $key }}" {
		{{if eq $value.Type "text" -}} 
		type = "{{$value.Type}}"
		value = <<EOV
{{$value.Value}}
EOV
		{{- else -}}
		type = "{{$value.Type}}"
		value = "{{$value.Value}}"
		{{- end}}
	} 
{{ end }}

pipelines = {
{{ range $key, $value := .Pipelines }}
    "{{ $key }}" {
        {{if .Triggers -}}
        triggers : {
            {{ range $key, $value := .Triggers }}
            "{{ $key }}" {
                {{if $value.ProjectKey -}} project_key: "{{ $value.ProjectKey }}" {{- end}}
                {{if $value.ApplicationName -}} application_name: "{{ $value.ApplicationName }}" {{- end}}
                {{if $value.FromEnvironment -}} from_environment: "{{ $value.FromEnvironment }}" {{- end}}
                {{if $value.ToEnvironment -}} to_environment: "{{ $value.ToEnvironment }}" {{- end}}
                manual: {{ $value.Manual }}
                {{range .Conditions -}}
                conditions {
                    variable: "{{ .Variable }}"
                    expected: "{{ .Expected }}"
                } 
                {{- end}}
            }
            {{ end }}
        }
        {{- end}}
        {{ range .Options }}
        options {
            {{if .Notifications -}}
            notifications {
                {{ range $key, $value := .Notifications -}}
                    "{{ $key }}" { {{ $value.JSON }} }
                {{- end}}
            }
            {{- end}}
            {{if .Environment -}} environment: "{{ .Environment }}" {{- end}}
            {{if .Hook -}} hook: "{{ .Hook }}" {{- end}}
            {{if .Polling -}} polling: "{{ .Polling }}" {{- end}}
            {{ range .Schedulers -}}
            schedulers {
                cron_expr: "{{.CronExpr}}"
                {{if .Parameters -}}
                parameters {
                    {{ range $key, $value := .Parameters }}
                    "{{ $key }}" {
                        {{if eq $value.Type "text" -}} 
                        type = "{{$value.Type}}"
                        value = <<EOV
{{$value.Value}}
EOV
                        {{- else -}}
                        type = "{{$value.Type}}"
                        value = "{{$value.Value}}"
                        {{- end}}
                    } 
                {{ end }}
                }
                {{- end}}
            }
            {{- end}}
        }
        {{- end}}       
    }
{{end }}
}
`
	t := template.New("t")
	return t.Parse(tmpl)
}
