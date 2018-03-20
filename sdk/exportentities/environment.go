package exportentities

import (
	"github.com/ovh/cds/sdk"
)

// Environment is a struct to export sdk.Environment
type Environment struct {
	Name        string                   `json:"name" yaml:"name"`
	Values      map[string]VariableValue `json:"values,omitempty" yaml:"values,omitempty"`
	Keys        map[string]KeyValue      `json:"keys,omitempty" yaml:"keys,omitempty"`
	Permissions map[string]int           `json:"permissions,omitempty" yaml:"permissions,omitempty"`
}

//NewEnvironment returns an Environment from an sdk.Environment pointer
func NewEnvironment(e sdk.Environment, withPermissions bool, keys []EncryptedKey) (env *Environment) {
	env = new(Environment)
	env.Name = e.Name
	env.Values = make(map[string]VariableValue, len(e.Variable))
	for _, v := range e.Variable {
		env.Values[v.Name] = VariableValue{
			Type:  string(v.Type),
			Value: v.Value,
		}
	}
	if withPermissions {
		env.Permissions = make(map[string]int, len(e.EnvironmentGroups))
		for _, p := range e.EnvironmentGroups {
			env.Permissions[p.Group.Name] = p.Permission
		}
	}
	env.Keys = make(map[string]KeyValue, len(keys))
	for _, k := range keys {
		env.Keys[k.Name] = KeyValue{
			Type:  k.Type,
			Value: k.Content,
		}
	}
	return
}

//Environment returns a sdk.Environment entity
func (e *Environment) Environment() (env *sdk.Environment) {
	env = new(sdk.Environment)
	env.Name = e.Name
	env.Variable = make([]sdk.Variable, len(e.Values))
	var i int
	for k, v := range e.Values {
		if v.Type == "" {
			v.Type = sdk.StringVariable
		}
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
