package exportentities

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

// Pipeline represents exported sdk.Pipeline
type Pipeline struct {
	Name        string                    `json:"name" yaml:"name"`
	Type        string                    `json:"type" yaml:"type"`
	Permissions map[string]int            `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Parameters  map[string]ParameterValue `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Stages      map[string]Stage          `json:"stages,omitempty" yaml:"stages,omitempty"`
	Jobs        map[string]Job            `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Steps       []Step                    `json:"steps,omitempty" yaml:"steps,omitempty" hcl:"step,omitempty"`
}

// Stage represents exported sdk.Stage
type Stage struct {
	Enabled    *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Jobs       map[string]Job    `json:"jobs,omitempty" yaml:"jobs,omitempty"`
	Conditions map[string]string `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// Job represents exported sdk.Job
type Job struct {
	Description  string        `json:"description,omitempty" yaml:"description,omitempty"`
	Enabled      *bool         `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Steps        []Step        `json:"steps,omitempty" yaml:"steps,omitempty" hcl:"step,omitempty"`
	Requirements []Requirement `json:"requirements,omitempty" yaml:"requirements,omitempty" hcl:"requirement,omitempty"`
}

// Step represents exported step used in a job
type Step struct {
	Enabled          *bool                        `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Final            *bool                        `json:"final,omitempty" yaml:"final,omitempty"`
	ArtifactUpload   map[string]string            `json:"artifactUpload,omitempty" yaml:"artifactUpload,omitempty"`
	ArtifactDownload map[string]string            `json:"artifactDownload,omitempty" yaml:"artifactDownload,omitempty"`
	Script           string                       `json:"script,omitempty" yaml:"script,omitempty"`
	JUnitReport      string                       `json:"jUnitReport,omitempty" yaml:"jUnitReport,omitempty"`
	Plugin           map[string]map[string]string `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Action           map[string]map[string]string `json:"action,omitempty" yaml:"action,omitempty"`
}

// Requirement represents an exported sdk.Requirement
type Requirement struct {
	Binary   string             `json:"binary,omitempty" yaml:"binary,omitempty"`
	Network  string             `json:"network,omitempty" yaml:"network,omitempty"`
	Model    string             `json:"model,omitempty" yaml:"model,omitempty"`
	Hostname string             `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	Plugin   string             `json:"plugin,omitempty" yaml:"plugin,omitempty"`
	Service  ServiceRequirement `json:"service,omitempty" yaml:"service,omitempty"`
	Memory   string             `json:"memory,omitempty" yaml:"memory,omitempty"`
}

// ServiceRequirement represents an exported sdk.Requirement of type ServiceRequirement
type ServiceRequirement struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}

//NewPipeline creates an exportable pipeline from a sdk.Pipeline
func NewPipeline(pip *sdk.Pipeline) (p *Pipeline) {
	p = &Pipeline{}
	p.Name = pip.Name
	p.Type = string(pip.Type)
	p.Permissions = make(map[string]int, len(pip.GroupPermission))
	for _, perm := range pip.GroupPermission {
		p.Permissions[perm.Group.Name] = perm.Permission
	}
	p.Parameters = make(map[string]ParameterValue, len(pip.Parameter))
	for _, v := range pip.Parameter {
		p.Parameters[v.Name] = ParameterValue{
			Type:         string(v.Type),
			DefaultValue: v.Value,
		}
	}

	switch len(pip.Stages) {
	case 0:
		return
	case 1:
		if len(pip.Stages[0].Prerequisites) == 0 {
			switch len(pip.Stages[0].Jobs) {
			case 0:
				return
			case 1:
				p.Steps = newSteps(pip.Stages[0].Jobs[0].Action)
				return
			default:
				p.Jobs = newJobs(pip.Stages[0].Jobs)
			}
			return
		}
		p.Stages = newStages(pip.Stages)
	default:
		p.Stages = newStages(pip.Stages)
	}

	return
}

func newStages(stages []sdk.Stage) map[string]Stage {
	res := map[string]Stage{}
	var order int
	for _, s := range stages {
		if len(s.Jobs) == 0 {
			continue
		}
		order++
		st := Stage{}
		if !s.Enabled {
			st.Enabled = &s.Enabled
		}
		if len(s.Prerequisites) > 0 {
			st.Conditions = make(map[string]string)
		}
		for _, r := range s.Prerequisites {
			st.Conditions[r.Parameter] = r.ExpectedValue
		}
		st.Jobs = newJobs(s.Jobs)
		res[fmt.Sprintf("%d|%s", order, s.Name)] = st
	}
	return res
}

func newJobs(jobs []sdk.Job) map[string]Job {
	res := map[string]Job{}
	for _, j := range jobs {
		if len(j.Action.Actions) == 0 {
			continue
		}
		jo := Job{}
		if !j.Enabled {
			jo.Enabled = &j.Enabled
		}
		jo.Steps = newSteps(j.Action)
		jo.Description = j.Action.Description
		for _, r := range j.Action.Requirements {
			switch r.Type {
			case sdk.BinaryRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Binary: r.Value})
			case sdk.NetworkAccessRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Network: r.Value})
			case sdk.ModelRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Model: r.Value})
			case sdk.HostnameRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Hostname: r.Value})
			case sdk.PluginRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Plugin: r.Value})
			case sdk.ServiceRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Service: ServiceRequirement{Name: r.Name, Value: r.Value}})
			case sdk.MemoryRequirement:
				jo.Requirements = append(jo.Requirements, Requirement{Memory: r.Value})
			}
		}

		res[j.Action.Name] = jo

	}
	return res
}

func newSteps(a sdk.Action) []Step {
	res := []Step{}
	for _, a := range a.Actions {
		s := Step{}
		if !a.Enabled {
			s.Enabled = &a.Enabled
		}
		if !a.Final {
			s.Final = &a.Final
		}

		switch a.Type {
		case sdk.BuiltinAction:
			switch a.Name {
			case sdk.ScriptAction:
				script := sdk.ParameterFind(a.Parameters, "script")
				if script != nil {
					s.Script = script.Value
				}
			case sdk.ArtifactDownload:
				s.ArtifactDownload = map[string]string{}
				path := sdk.ParameterFind(a.Parameters, "path")
				if path != nil {
					s.ArtifactDownload["path"] = path.Value
				}
				tag := sdk.ParameterFind(a.Parameters, "tag")
				if tag != nil {
					s.ArtifactDownload["tag"] = tag.Value
				}
			case sdk.ArtifactUpload:
				s.ArtifactUpload = map[string]string{}
				path := sdk.ParameterFind(a.Parameters, "path")
				if path != nil {
					s.ArtifactUpload["path"] = path.Value
				}
				tag := sdk.ParameterFind(a.Parameters, "tag")
				if tag != nil {
					s.ArtifactUpload["tag"] = tag.Value
				}
			case sdk.JUnitAction:
				path := sdk.ParameterFind(a.Parameters, "path")
				if path != nil {
					s.JUnitReport = path.Value
				}
			}
		case sdk.PluginAction:
			s.Plugin = map[string]map[string]string{}
			s.Plugin[a.Name] = map[string]string{}
			for _, p := range a.Parameters {
				if p.Value != "" {
					s.Plugin[a.Name][p.Name] = p.Value
				}
			}
		default:
			s.Action = map[string]map[string]string{}
			s.Action[a.Name] = map[string]string{}
			for _, p := range a.Parameters {
				if p.Value != "" {
					s.Action[a.Name][p.Name] = p.Value
				}
			}
		}

		res = append(res, s)
	}

	return res
}

//HCLTemplate returns text/template
/*
func (p *Pipeline) HCLTemplate() (*template.Template, error) {
	tmpl := `name = "{{.Name}}"
type = "{{.Type}}"
{{if .Permissions -}}
permissions  { {{ range $key, $value := .Permissions }}
	"{{$key}}" = {{$value}}{{ end }}
}
{{- end}}
{{if .Steps -}}
{{ range $value := .Steps}}
step {
	{{if $value.Enabled -}} enabled = {{ $value.Enabled }}{{- end}} 	{{if $value.Final -}} final = {{ $value.Final }}{{- end}}
	{{if .Script -}}  script = "{{.Script}}"
	{{- else if .Action -}}
		action {
			{{ range $key, $value := .Action }}
			{{$key}} {
				{{ range $key, $value := $value }}
				{{$key}} = "{{$value}}"{{end}}
			}
			{{end}}}
		}
	{{- else if .Plugin -}}
		plugin {
			{{ range $key, $value := .Plugin }}
			{{$key}} {
				{{ range $key, $value := $value }}
				{{$key}} = "{{$value}}"{{end}}
			}
			{{end}}}
		}
	{{- else if .ArtifactUpload -}}
		artifactUpload {
		{{ range $key, $value := .ArtifactUpload }}
			{{$key}} = "{{$value}}"{{ end }}
		}
	{{- else if .ArtifactDownload -}}
		artifactDownload {
		{{ range $key, $value := .ArtifactDownload }}
			{{$key}} = "{{$value}}"{{ end }}
		}
	{{- else if .JUnitReport -}}
		jUnitReport = "{{.JUnitReport}}"
	{{- end}}
}
{{ end }}

{{- else -}}

{{if .Jobs -}}
{{ range $key, $value := .Jobs }}
job "{{$key}}" {
	{{ range $key, $value := $value.Steps}}step {
		{{if $value.Enabled -}} enabled = {{ $value.Enabled }}{{- end}}
		{{if $value.Final -}} final = {{ $value.Final }}{{- end}}
		{{if .Script -}}
			script = <<EOV
{{.Script}}
EOV
		{{- else if .Action -}}
			action {
				{{ range $key, $value := .Action }}
				{{$key}} {
					{{ range $key, $value := $value }}
					{{$key}} = "{{$value}}"{{end}}
				}
				{{end}}}
			}
		{{- else if .Plugin -}}
			plugin {
				{{ range $key, $value := .Plugin }}
				{{$key}} {
					{{ range $key, $value := $value }}
					{{$key}} = "{{$value}}"{{end}}
				}
				{{end}}}
			}
		{{- else if .ArtifactUpload -}}
			artifactUpload {
			{{ range $key, $value := .ArtifactUpload }}
				{{$key}} = "{{$value}}"{{ end }}
			}
		{{- else if .ArtifactDownload -}}
			artifactDownload {
			{{ range $key, $value := .ArtifactDownload}}
				{{$key}} = "{{$value}}"{{ end }}
			}
		{{- else if .JUnitReport -}}
			jUnitReport = "{{.JUnitReport}}"
		{{- end}}
	}
	{{ end }}
}
{{ end }}
{{- else -}}
stages {
{{ range $key, $value := .Stages }}
	stage "{{$key}}" {
		{{ range $key, $value := $value.Jobs }}
		job "{{$key}}" {
			{{ range $key, $value := $value.Steps}}step {
				{{if $value.Enabled -}} enabled = {{ $value.Enabled }}{{- end}}
				{{if $value.Final -}} final = {{ $value.Final }}{{- end}}
				{{if .Script -}}
					script = <<EOV
{{.Script}}
EOV
				{{- else if .Action -}}
					action {
						{{ range $key, $value := .Action }}
						{{$key}} {
							{{ range $key, $value := $value }}
							{{$key}} = "{{$value}}"{{end}}
						}
						{{end}}}
					}
				{{- else if .Plugin -}}
					plugin {
						{{ range $key, $value := .Plugin }}
						{{$key}} {
							{{ range $key, $value := $value }}
							{{$key}} = "{{$value}}"{{end}}
						}
						{{end}}}
					}
				{{- else if .ArtifactUpload -}}
					artifactUpload {
					{{ range $key, $value := .ArtifactUpload }}
						{{$key}} = "{{$value}}"{{ end }}
					}
				{{- else if .ArtifactDownload -}}
					artifactDownload {
					{{ range $key, $value := .ArtifactDownload -}}
						{{$key}} = "{{$value}}"{{- end }}
					}
				{{- else if .JUnitReport -}}
					jUnitReport = "{{.JUnitReport}}"
				{{- end}}
			}
			{{ end }}
		}
		{{ end }}
	}
{{ end}}}
{{- end}}
{{- end}}
`
	t := template.New("t")
	return t.Parse(tmpl)
}

func decodePipeline(m map[string]interface{}) (p *Pipeline, err error) {
	if m["name"] == nil || m["type"] == nil {
		err = errors.New("Invalid pipeline structured map")
		return
	}

	p = &Pipeline{
		Name: m["name"].(string),
		Type: m["type"].(string),
	}

	if m["step"] != nil {
		steps := m["step"].([]map[string]interface{})
		p.Steps = make([]Step, len(steps))
		for i, s := range steps {
			p.Steps[i] = Step{}
			p.Steps[i].Enabled = getBoolPtr(s, "enabled")
			p.Steps[i].Final = getBoolPtr(s, "final")
			p.Steps[i].Script = getStringValue(s, "script")
			p.Steps[i].Action = getMapStringMapStringString(s, "action")
			p.Steps[i].Plugin = getMapStringMapStringString(s, "plugin")
			p.Steps[i].ArtifactDownload = getMapStringString(s, "artifactDownload")
			p.Steps[i].ArtifactUpload = getMapStringString(s, "artifactUpload")
			p.Steps[i].JUnitReport = getStringValue(s, "jUnitReport")
		}
	}

	return
}

func getBoolPtr(m map[string]interface{}, k string) *bool {
	if m[k] == nil {
		return nil
	}
	b, ok := m[k].(bool)
	if !ok {
		return nil
	}
	return &b
}

func getStringValue(m map[string]interface{}, k string) (v string) {
	if m[k] == nil {
		return
	}
	var ok bool
	v, ok = m[k].(string)
	if !ok {
		v = ""
	}
	return
}

func getMapStringString(m map[string]interface{}, k string) (r map[string]string) {
	if m[k] == nil {
		return nil
	}
	var ok bool
	r, ok = m[k].(map[string]string)
	if !ok {
		return nil
	}
	return
}

func getMapStringMapStringString(m map[string]interface{}, k string) (r map[string]map[string]string) {
	if m[k] == nil {
		return nil
	}
	var ok bool
	r, ok = m[k].(map[string]map[string]string)
	if !ok {
		return nil
	}
	return
}
*/
