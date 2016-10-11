package sdk

import (
	"fmt"
	"strings"
)

// ParameterType defines the types of parameter of a pipeline or a action
type ParameterType string

// Different type of Parameter
const (
	EnvironmentParameter ParameterType = "env"
	PipelineParameter    ParameterType = "pipeline"
	ListParameter        ParameterType = "list"
	NumberParameter      ParameterType = "number"
	StringParameter      ParameterType = "string"
	TextParameter        ParameterType = "text"
	BooleanParameter     ParameterType = "boolean"
)

var (
	// AvailableParameterType list all existing parameters type in CDS
	AvailableParameterType = []string{
		string(StringParameter),
		string(NumberParameter),
		string(TextParameter),
		string(EnvironmentParameter),
		string(BooleanParameter),
		string(ListParameter),
		string(PipelineParameter),
	}
)

// Value of passwords when leaving the API
const (
	PasswordPlaceholder string = "**********"
)

// ParameterTypeFromString returns a parameter Type from a given string
func ParameterTypeFromString(in string) ParameterType {
	switch in {
	case EnvironmentParameter.String():
		return EnvironmentParameter
	case ListParameter.String():
		return ListParameter
	case NumberParameter.String():
		return NumberParameter
	case StringParameter.String():
		return StringParameter
	case TextParameter.String():
		return TextParameter
	case BooleanParameter.String():
		return BooleanParameter
	case PipelineParameter.String():
		return PipelineParameter
	default:
		return TextParameter
	}
}

func (t ParameterType) String() string {
	return string(t)
}

// Parameter can be a String/Date/Script/URL...
type Parameter struct {
	ID          int64         `json:"id" yaml:"-"`
	Name        string        `json:"name"`
	Type        ParameterType `json:"type"`
	Value       string        `json:"value"`
	Description string        `json:"description" yaml:"desc,omitempty"`
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
