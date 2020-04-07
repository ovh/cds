package sdk

// SchemaResponse contains all json schema for a user.
type SchemaResponse struct {
	Workflow    string `json:"workflow"`
	Application string `json:"application"`
	Pipeline    string `json:"pipeline"`
	Environment string `json:"environment"`
}
