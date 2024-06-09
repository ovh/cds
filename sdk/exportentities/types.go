package exportentities

type (
	//Format is a type
	Format int

	// VariableValue is a struct to export a value of Variable
	VariableValue struct {
		Type  string `json:"type,omitempty" yaml:"type,omitempty"`
		Value string `json:"value,omitempty" yaml:"value,omitempty"`
	}

	// KeyValue is a struct to export a value of Key
	KeyValue struct {
		Type  string `json:"type,omitempty" yaml:"type,omitempty"`
		Value string `json:"value,omitempty" yaml:"value,omitempty"`
		Regen *bool  `json:"regen,omitempty" yaml:"regen,omitempty"`
	}

	// ParameterValue is a struct to export a default value of Parameter
	ParameterValue struct {
		Type         string `json:"type,omitempty" yaml:"type,omitempty"`
		DefaultValue string `json:"default,omitempty" yaml:"default,omitempty"`
		Description  string `json:"description,omitempty" yaml:"description,omitempty"`
		Advanced     *bool  `json:"advanced,omitempty" yaml:"advanced,omitempty"`
	}
)

// All the consts
const (
	FormatJSON Format = iota
	FormatYAML
	UnknownFormat
)

func (f Format) ContentType() string {
	switch f {
	case FormatYAML:
		return "application/x-yaml"
	case FormatJSON:
		return "application/json"
	}
	return "application/octet-stream"
}
