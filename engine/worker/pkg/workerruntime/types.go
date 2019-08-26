package workerruntime

import (
	"context"
	"errors"
	"fmt"

	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/afero"
)

type DownloadArtifact struct {
	Workflow    string `json:"workflow"`
	Number      int64  `json:"number"`
	Pattern     string `json:"pattern" cli:"pattern"`
	Tag         string `json:"tag" cli:"tag"`
	Destination string `json:"destination"`
}

type FilePath struct {
	Path string `json:"path"`
}

type KeyResponse struct {
	PKey string `json:"pkey"`
	Type string `json:"type"`
}

type TmplPath struct {
	Path        string `json:"path"`
	Destination string `json:"destination"`
}

type Level string

type (
	contextKey int
)

const (
	jobID contextKey = iota
	stepOrder
	workDir
	keysDir
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

type Runtime interface {
	Name() string
	Register(ctx context.Context) error
	Take(ctx context.Context, job sdk.WorkflowNodeJobRun) error
	ProcessJob(job sdk.WorkflowNodeJobRunData) (sdk.Result, error)
	SendLog(ctx context.Context, level Level, format string)
	InstallKey(key sdk.Variable, destinationPath string) (*KeyResponse, error)
	Unregister() error
	Client() cdsclient.WorkerInterface
	Workspace() afero.Fs
	Environ() []string
	Blur(interface{}) error
	HTTPPort() int32
}

func JobID(ctx context.Context) (int64, error) {
	jobIDStr := ctx.Value(jobID)
	jobID, ok := jobIDStr.(int64)
	if !ok {
		return -1, errors.New("unable to get job ID")
	}
	return jobID, nil
}

func SetJobID(ctx context.Context, i int64) context.Context {
	return context.WithValue(ctx, jobID, i)
}

func StepOrder(ctx context.Context) (int, error) {
	stepOrderStr := ctx.Value(stepOrder)
	stepOrder, ok := stepOrderStr.(int)
	if !ok {
		return -1, fmt.Errorf("unable to get step order: got %v", stepOrder)
	}
	return stepOrder, nil
}

func SetStepOrder(ctx context.Context, i int) context.Context {
	return context.WithValue(ctx, stepOrder, i)
}

func WorkingDirectory(ctx context.Context) (afero.File, error) {
	wdi := ctx.Value(workDir)
	wd, ok := wdi.(afero.File)
	if !ok {
		return nil, errors.New("unable to get working directory")
	}
	log.Debug("WorkingDirectory> working directory is : %s", wd.Name())
	return wd, nil
}

func SetWorkingDirectory(ctx context.Context, s afero.File) context.Context {
	log.Debug("SetWorkingDirectory> working directory is: %s", s.Name())
	return context.WithValue(ctx, workDir, s)
}

func KeysDirectory(ctx context.Context) (afero.File, error) {
	wdi := ctx.Value(keysDir)
	wd, ok := wdi.(afero.File)
	if !ok {
		return nil, errors.New("unable to get working directory")
	}
	log.Debug("KeysDirectory> working directory is : %s", wd.Name())
	return wd, nil
}

func SetKeysDirectory(ctx context.Context, s afero.File) context.Context {
	log.Debug("SetKeysDirectory> working directory is: %s", s.Name())
	return context.WithValue(ctx, keysDir, s)
}
