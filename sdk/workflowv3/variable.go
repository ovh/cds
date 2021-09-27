package workflowv3

import (
	"reflect"

	"github.com/ovh/cds/sdk"
)

type Variables map[string]Variable

func (v *Variables) UnmarshalYAML(unmarshal func(interface{}) error) error {
	raw := make(map[string]Variable)
	if err := unmarshal(&raw); err != nil {
		return err
	}
	res := make(Variables)
	for k, variable := range raw {
		value := reflect.ValueOf(variable)
		switch value.Kind() {
		case reflect.Map:
			m := make(map[string]interface{})
			rawVariable, ok := variable.(map[interface{}]interface{})
			if !ok {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given variable %q", k)
			}
			for varKey, varValue := range rawVariable {
				varKeyString, ok := varKey.(string)
				if !ok {
					return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given variable %q", k)
				}
				m[varKeyString] = varValue
			}
			res[k] = m
		case reflect.String:
			res[k] = variable
		default:
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given variable %q", k)
		}
	}
	*v = res
	return nil
}

func (v Variables) ExistVariable(variableName string) bool {
	_, ok := v[variableName]
	return ok
}

type Variable interface{}
