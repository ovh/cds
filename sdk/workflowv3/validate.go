package workflowv3

type ValidationResponse struct {
	Valid                bool                 `json:"valid,omitempty"`
	Error                string               `json:"error,omitempty"`
	Workflow             Workflow             `json:"workflow,omitempty"`
	ExternalDependencies ExternalDependencies `json:"external_dependencies,omitempty"`
}
