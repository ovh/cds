package artifact_manager

import (
	"fmt"

	arti "github.com/ovh/cds/engine/api/integration/artifact_manager/artifactory"
	"github.com/ovh/cds/sdk"
)

type ArtifactManager interface {
	GetFileInfo(repoName string, filePath string) (sdk.FileInfo, error)
	SetProperties(repoName string, filePath string, values ...sdk.KeyValues) error
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
	return nil, fmt.Errorf("artifact Manager %s not implemented", managerType)
}
