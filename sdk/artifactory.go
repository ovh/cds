package sdk

import (
	"fmt"
	"os"
	"strings"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/config"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

func NewArtifactoryClient(url string, token string) (artifactory.ArtifactoryServicesManager, error) {
	log.SetLogger(log.NewLogger(log.INFO, os.Stdout))
	rtDetails := auth.NewArtifactoryDetails()
	// url must have a '/' at the end. We ensure to have this '/' (and only one)
	rtDetails.SetUrl(strings.TrimSuffix(url, "/") + "/")
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
		return nil, WrapError(err, "unable to create artifactory client")
	}
	return asm, nil
}
