package sdk

// Schema is the string representation of the JSON schema
type Schema []byte

// SchemaResponse contains all json schema for a user.
type SchemaResponse struct {
	Workflow    string `json:"workflow"`
	Application string `json:"application"`
	Pipeline    string `json:"pipeline"`
	Environment string `json:"environment"`
}
