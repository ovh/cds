package sdk

type FavoriteParams struct {
	Type         string `json:"type"`
	ProjectKey   string `json:"project_key"`
	WorkflowName string `json:"workflow_name,omitempty"`
}
