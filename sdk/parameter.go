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
func ParameterFind(vars []Parameter, s string) *Parameter {
	for _, v := range vars {
		if v.Name == s {
			return &v
		}
	}
	return nil
}

// ParametersFromMap returns an array of parameters from a map
func ParametersFromMap(m map[string]string) []Parameter {
	res := []Parameter{}
	for k, v := range m {
		res = append(res, Parameter{Name: k, Value: v, Type: "string"})
	}
	return res
}
