package artifact_manager

import (
	"fmt"
	"os"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/config"

	"github.com/jfrog/jfrog-client-go/utils/log"
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
		return newArtifactoryClient(url, token)
	}
	return nil, fmt.Errorf("artifact Manager %s not implemented", managerType)
}

func newArtifactoryClient(url string, token string) (ArtifactManager, error) {
	log.SetLogger(log.NewLogger(log.INFO, os.Stdout))
	rtDetails := auth.NewArtifactoryDetails()
	rtDetails.SetUrl(url)
	rtDetails.SetAccessToken(token)
	serviceConfig, err := config.NewConfigBuilder().
		SetServiceDetails(rtDetails).
		SetThreads(1).
		SetDryRun(false).
		Build()
	if err != nil {
		return nil, fmt.Errorf("unable to create service config: %v", err)
	}
	asm, err := artifactory.New(serviceConfig)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to create artifactory client")
	}
	return &arti.Client{Asm: asm}, nil
}
