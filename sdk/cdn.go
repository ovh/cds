package sdk

type CDNObjectType string

const (
	CDNArtifactType CDNObjectType = "CDNArtifactType"
)

type CDNRequest struct {
	ServiceName     string                   `json:"service_name" yaml:"service_name"`
	Type            CDNObjectType            `json:"type" yaml:"type"`
	ProjectKey      string                   `json:"project_key,omitempty" yaml:"project_key,omitempty"`
	IntegrationName string                   `json:"integration_name,omitempty" yaml:"integration_name,omitempty"`
	Config          map[string]string        `json:"config,omitempty" yaml:"config,omitempty"`
	Artifact        *WorkflowNodeRunArtifact `json:"artifact,omitempty" yaml:"artifact,omitempty"`
}
