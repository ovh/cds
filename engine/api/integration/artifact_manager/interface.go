package artifact_manager

import (
	"fmt"
	"os"

	"github.com/jfrog/jfrog-client-go/artifactory/buildinfo"
	"github.com/jfrog/jfrog-client-go/artifactory/services"
	"github.com/jfrog/jfrog-client-go/utils/log"
	arti "github.com/ovh/cds/engine/api/integration/artifact_manager/artifactory"

	"github.com/ovh/cds/sdk"
)

// mockgen -source=interface.go -package mock_artifact_manager -destination=mock_artifact_manager/interface_mock.go ArtifactManager
type ArtifactManager interface {
	GetFileInfo(repoName string, filePath string) (sdk.FileInfo, error)
	SetProperties(repoName string, filePath string, values ...sdk.KeyValues) error
	DeleteBuild(project string, buildName string, buildVersion string) error
	PublishBuildInfo(project string, request *buildinfo.BuildInfo) error
	XrayScanBuild(params services.XrayScanParams) ([]byte, error)
	GetURL() string
}

type ClientFactoryFunc func(string, string, string) (ArtifactManager, error)

var DefaultClientFactory ClientFactoryFunc = newClient

func NewClient(managerType, url, token string) (ArtifactManager, error) {
	return DefaultClientFactory(managerType, url, token)
}

func newClient(managerType, url, token string) (ArtifactManager, error) {
	log.SetLogger(log.NewLogger(log.INFO, os.Stdout))
	switch managerType {
	case "artifactory":
		asm, err := sdk.NewArtifactoryClient(url, token)
		if err != nil {
			return nil, err
		}
		return &arti.Client{Asm: asm}, nil
	}
	return nil, fmt.Errorf("artifact Manager %s not implemented", managerType)
}
