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
	switch log.Factory().GetLevel() {
	case log.LevelDebug:
		return rtlog.DEBUG
	case log.LevelInfo:
		return rtlog.INFO
	case log.LevelWarn:
		return rtlog.WARN
	case log.LevelError, log.LevelFatal, log.LevelPanic:
		return rtlog.ERROR
	default:
		return rtlog.INFO
	}
}

func (artifactoryLogger) SetLogLevel(rtlog.LevelType)      {}
func (artifactoryLogger) SetOutputWriter(writer io.Writer) {}
func (artifactoryLogger) SetLogsWriter(writer io.Writer)   {}

func (l artifactoryLogger) Debug(a ...interface{}) {
	log.Debug(context.Background(), l.BuildMsg(a))
}
func (l artifactoryLogger) Info(a ...interface{}) {
	log.Info(context.Background(), l.BuildMsg(a...))
}
func (l artifactoryLogger) Warn(a ...interface{}) {
	log.Warn(context.Background(), l.BuildMsg(a...))
}
func (l artifactoryLogger) Error(a ...interface{}) {
	log.Error(context.Background(), l.BuildMsg(a...))
}
func (l artifactoryLogger) Output(a ...interface{}) {
	log.Info(context.Background(), l.BuildMsg(a...))
}

func (l artifactoryLogger) BuildMsg(a ...interface{}) string {
	msg := make([]string, 0, len(a))
	for _, m := range a {
		msg = append(msg, fmt.Sprint(m))
	}
	return strings.Join(msg, " ")
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
		return nil, WrapError(err, "unable to create service config")
	}
	asm, err := artifactory.New(serviceConfig)
	if err != nil {
		return nil, WrapError(err, "unable to create artifactory client")
	}
	return asm, nil
}
