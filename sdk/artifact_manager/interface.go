package artifact_manager

import (
	"context"

	buildinfo "github.com/jfrog/build-info-go/entities"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	arti "github.com/ovh/cds/sdk/artifact_manager/artifactory"

	"github.com/ovh/cds/sdk"
)

// mockgen -source=interface.go -package mock_artifact_manager -destination=mock_artifact_manager/interface_mock.go ArtifactManager
type ArtifactManager interface {
	GetFileInfo(repoName string, filePath string) (sdk.FileInfo, error)
	GetRepository(repoName string) (*services.RepositoryDetails, error)
	GetFolderInfo(repoName string, folderPath string) (*utils.FolderInfo, error)
	GetProperties(repoName string, filePath string) (map[string][]string, error)
	SetProperties(repoName string, filePath string, values *utils.Properties) error
	DeleteBuild(project string, buildName string, buildVersion string) error
	PublishBuildInfo(project string, request *buildinfo.BuildInfo) error
	XrayScanBuild(params services.XrayScanParams) ([]byte, error)
	GetURL() string
	CheckArtifactExists(repoName string, artiName string) (bool, error)
	PromoteDocker(params services.DockerPromoteParams) error
	Copy(params services.MoveCopyParams) (successCount, failedCount int, err error)
	Move(params services.MoveCopyParams) (successCount, failedCount int, err error)
	GetRepositoryMaturity(repoName string) (string, error)
	Search(ctx context.Context, query string) (sdk.ArtifactResults, error)
}

type ClientFactoryFunc func(string, string, string) (ArtifactManager, error)

var DefaultClientFactory ClientFactoryFunc = newClient

func NewClient(managerType, url, token string) (ArtifactManager, error) {
	return DefaultClientFactory(managerType, url, token)
}

func newClient(managerType, url, token string) (ArtifactManager, error) {
	switch managerType {
	case "artifactory":
		asm, err := sdk.NewArtifactoryClient(url, token)
		if err != nil {
			return nil, err
		}
		return &arti.Client{Asm: asm}, nil
	}
	return nil, sdk.Errorf("artifact Manager %s not implemented", managerType)
}
