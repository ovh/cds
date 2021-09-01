package sdk

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/rockbears/log"

	"github.com/jfrog/jfrog-client-go/artifactory"
	"github.com/jfrog/jfrog-client-go/artifactory/auth"
	"github.com/jfrog/jfrog-client-go/config"
	rtlog "github.com/jfrog/jfrog-client-go/utils/log"
)

type artifactoryLogger struct{}

func (artifactoryLogger) GetLogLevel() rtlog.LevelType {
	return rtlog.LevelType(log.Factory().GetLevel())
}

func (artifactoryLogger) SetLogLevel(rtlog.LevelType)      {}
func (artifactoryLogger) SetOutputWriter(writer io.Writer) {}
func (artifactoryLogger) SetLogsWriter(writer io.Writer)   {}

func (l artifactoryLogger) Debug(a ...interface{}) {
	log.Debug(context.Background(), fmt.Sprint(a...))
}
func (l artifactoryLogger) Info(a ...interface{}) {
	log.Info(context.Background(), fmt.Sprint(a...))
}
func (l artifactoryLogger) Warn(a ...interface{}) {
	log.Warn(context.Background(), fmt.Sprint(a...))
}
func (l artifactoryLogger) Error(a ...interface{}) {
	log.Error(context.Background(), fmt.Sprint(a...))
}
func (l artifactoryLogger) Output(a ...interface{}) {
	log.Info(context.Background(), fmt.Sprint(a...))
}

func NewArtifactoryClient(url string, token string) (artifactory.ArtifactoryServicesManager, error) {
	logger := artifactoryLogger{}
	rtlog.SetLogger(logger)
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
