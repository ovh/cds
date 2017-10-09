package sdk

// Usage is type to represent usage of each type
type Usage struct {
	Workflows    []Workflow    `json:"workflows,omitempty"`
	Environments []Environment `json:"environments,omitempty"`
	Pipelines    []Pipeline    `json:"pipelines,omitempty"`
	Applications []Application `json:"applications,omitempty"`
}
