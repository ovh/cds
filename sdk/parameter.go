package sdk

import (
	"fmt"
	"strings"
)

// Different type of Parameter
const (
	EnvironmentParameter = "env"
	PipelineParameter    = "pipeline"
	ListParameter        = "list"
	NumberParameter      = "number"
	StringParameter      = "string"
	TextParameter        = "text"
	BooleanParameter     = "boolean"
	KeyParameter         = "key"
)

var (
	// AvailableParameterType list all existing parameters type in CDS
	AvailableParameterType = []string{
		StringParameter,
		NumberParameter,
		TextParameter,
		EnvironmentParameter,
		BooleanParameter,
		ListParameter,
		PipelineParameter,
		KeyParameter,
	}
)

// Value of passwords when leaving the API
const (
	PasswordPlaceholder string = "**********"
)

// Parameter can be a String/Date/Script/URL...
type Parameter struct {
	ID          int64  `json:"id" yaml:"-"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Value       string `json:"value"`
	Description string `json:"description" yaml:"desc,omitempty"`
}

// NewStringParameter creates a Parameter from a string with <name>=<value> format
func NewStringParameter(s string) (Parameter, error) {
	var p Parameter

	t := strings.SplitN(s, "=", 2)
	if len(t) != 2 {
		return p, fmt.Errorf("cds: wrong format parameter")
	}
	p.Name = t[0]
	p.Type = StringParameter
	p.Value = t[1]

	return p, nil
}

// AddParameter append a parameter in a parameter array
func AddParameter(array *[]Parameter, name string, parameterType string, value string) {
	params := append(*array, Parameter{
		Name:  name,
		Value: value,
		Type:  parameterType,
	})
	*array = params
}

// ParameterFind return a parameter given its name if it exists in array
func ParameterFind(vars *[]Parameter, s string) *Parameter {
	for i, v := range *vars {
		if v.Name == s {
			return &(*vars)[i]
		}
	}
	return nil
}

// ParameterValue return a parameter value given its name if it exists in array, else it returns empty string
func ParameterValue(vars []Parameter, s string) string {
	p := ParameterFind(&vars, s)
	if p == nil {
		return ""
	}
	return p.Value
}

// ParametersFromMap returns an array of parameters from a map
func ParametersFromMap(m map[string]string) []Parameter {
	res := []Parameter{}
	for k, v := range m {
		res = append(res, Parameter{Name: k, Value: v, Type: "string"})
	}
	return res
}

// ParametersToMap returns a map from a slice of parameters
func ParametersToMap(params []Parameter) map[string]string {
	res := map[string]string{}
	for _, p := range params {
		res[p.Name] = p.Value
	}
	return res
}

// ParametersFromProjectVariables returns a map from a slice of parameters
func ParametersFromProjectVariables(proj *Project) map[string]string {
	if proj == nil {
		return nil
	}
	params := variablesToParameters("cds.proj", proj.Variable)
	return ParametersToMap(params)
}

// ParametersFromApplicationVariables returns a map from a slice of parameters
func ParametersFromApplicationVariables(app *Application) map[string]string {
	if app == nil {
		return nil
	}
	params := variablesToParameters("cds.app", app.Variable)
	return ParametersToMap(params)
}

// ParametersFromEnvironmentVariables returns a map from a slice of parameters
func ParametersFromEnvironmentVariables(env *Environment) map[string]string {
	if env == nil {
		return nil
	}
	params := variablesToParameters("cds.env", env.Variable)
	return ParametersToMap(params)
}

// ParametersFromPipelineParameters returns a map from a slice of parameters
func ParametersFromPipelineParameters(pipParams []Parameter) map[string]string {
	res := []Parameter{}
	for _, t := range pipParams {
		t.Name = "cds.pip." + t.Name
		res = append(res, Parameter{Name: t.Name, Type: t.Type, Value: t.Value})
	}
	return ParametersToMap(res)
}

func variablesToParameters(prefix string, variables []Variable) []Parameter {
	res := []Parameter{}
	for _, t := range variables {
		if NeedPlaceholder(t.Type) {
			continue
		}
		t.Name = prefix + "." + t.Name
		res = append(res, Parameter{Name: t.Name, Type: t.Type, Value: t.Value})
	}
	return res
}

// ParametersMapMerge merges two maps of parameters preserving all values
func ParametersMapMerge(params map[string]string, otherParams map[string]string) map[string]string {
	for k, v := range otherParams {
		if val, ok := params[k]; ok {
			if val != v {
				if val == "" { // val empty, take v, even if v is empty
					params[k] = fmt.Sprintf("%s", v)
				} else { // take val, if v is empty or not
					params[k] = fmt.Sprintf("%s", val)
				}
				continue
			}
			continue
		}
		params[k] = v
	}
	return params
}
