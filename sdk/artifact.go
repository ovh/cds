package sdk

// Builtin artifact manipulation actions
const (
	ArtifactUpload   = "Artifact Upload"
	ArtifactDownload = "Artifact Download"

	ArtifactUploadPluginInputPath = "cds.integration.artifact_manager.upload.path"

	ArtifactUploadPluginOutputPathFileName = "name"
	ArtifactUploadPluginOutputPathFilePath = "path"
	ArtifactUploadPluginOutputPathRepoType = "repository_type"
	ArtifactUploadPluginOutputPathRepoName = "repository_name"
	ArtifactUploadPluginOutputPathMD5      = "md5"
	ArtifactUploadPluginOutputPerm         = "perm"
	ArtifactUploadPluginOutputSize         = "size"
	ArtifactUploadPluginOutputFileType     = "file_type"

	ArtifactDownloadPluginInputDestinationPath = "cds.integration.artifact_manager.download.destination.path"
	ArtifactDownloadPluginInputFilePath        = "cds.integration.artifact_manager.download.file.path"
	ArtifactDownloadPluginInputMd5             = "cds.integration.artifact_manager.download.file.md5"
	ArtifactDownloadPluginInputPerm            = "cds.integration.artifact_manager.download.file.perm"

	ArtifactFileTypeCoverage = "coverage"
)

// ArtifactsStore represents
type ArtifactsStore struct {
	Name                  string `json:"name"`
	TemporaryURLSupported bool   `json:"temporary_url_supported"`
}
