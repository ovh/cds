package sdk

// Builtin artifact manipulation actions
const (
	ArtifactUpload   = "Artifact Upload"
	ArtifactDownload = "Artifact Download"
	ServeStaticFiles = "Serve Static Files"
)

// ArtifactsStore represents
type ArtifactsStore struct {
	Name                  string `json:"name"`
	Private               bool   `json:"private"`
	TemporaryURLSupported bool   `json:"temporary_url_supported"`
}
