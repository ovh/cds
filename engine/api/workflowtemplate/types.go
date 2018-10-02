package workflowtemplate

// ParameterType used for template parameter.
type ParameterType string

// Parameter types.
const (
	String  ParameterType = "string"
	Boolean ParameterType = "boolean"
)

// Parameter for template.
type Parameter struct {
	Key      string        `json:"key"`
	Type     ParameterType `json:"type"`
	Required bool          `json:"required"`
}

// Template struct.
type Template struct {
	Parameters []Parameter `json:"parameters"`
	Workflow   string      `json:"workflow"`
	Pipelines  []string    `json:"pipelines"`
}

// Request struct use for execution request.
type Request struct {
	Name       string `json:"name"`
	Parameters map[string]string
}

// Result struct.
type Result struct {
	Workflow  string   `json:"workflow"`
	Pipelines []string `json:"pipelines"`
}
