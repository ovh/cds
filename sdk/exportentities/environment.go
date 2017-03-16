package exportentities

import (
	"text/template"

	"github.com/ovh/cds/sdk"
)

// Environment is a struct to export sdk.Environment
type Environment struct {
	Name        string                   `json:"name" yaml:"name"`
	Values      map[string]VariableValue `json:"values" yaml:"values"`
	Permissions map[string]int           `json:"permissions" yaml:"permissions"`
}

//NewEnvironment returns an Environment from an sdk.Environment pointer
func NewEnvironment(e *sdk.Environment) (env *Environment) {
	if e == nil {
		return
	}
	env = new(Environment)
	env.Name = e.Name
	env.Values = make(map[string]VariableValue, len(e.Variable))
	for _, v := range e.Variable {
		env.Values[v.Name] = VariableValue{
			Type:  string(v.Type),
			Value: v.Value,
		}
	}
	env.Permissions = make(map[string]int, len(e.EnvironmentGroups))
	for _, p := range e.EnvironmentGroups {
		env.Permissions[p.Group.Name] = p.Permission
	}
	return
}

//HCLTemplate returns text/template
func (e *Environment) HCLTemplate() (*template.Template, error) {
	tmpl := `name = "{{.Name}}"

permissions = { {{ range $key, $value := .Permissions }}
	"{{$key}}" = {{$value}}{{ end }}
}

values = { 
{{ range $key, $value := .Values }}
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
}`
	t := template.New("t")
	return t.Parse(tmpl)
}

//Environment returns a sdk.Environment entity
func (e *Environment) Environment() (env *sdk.Environment) {
	env = new(sdk.Environment)
	env.Name = e.Name
	env.Variable = make([]sdk.Variable, len(e.Values))
	var i int
	for k, v := range e.Values {
		env.Variable[i] = sdk.Variable{
			Name:  k,
			Type:  v.Type,
			Value: v.Value,
		}
		i++
	}
	env.EnvironmentGroups = make([]sdk.GroupPermission, len(e.Permissions))
	i = 0
	for k, v := range e.Permissions {
		env.EnvironmentGroups[i] = sdk.GroupPermission{
			Group: sdk.Group{
				Name: k,
			},
			Permission: v,
		}
		i++
	}

	return
}
