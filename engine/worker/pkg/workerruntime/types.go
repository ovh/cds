package workerruntime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ovh/cds/sdk/cdsclient"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/afero"
)

type WorkerConfig struct {
	Name                     string            `json:"name"`
	Basedir                  string            `json:"basedir"`
	Log                      cdslog.Conf       `json:"log"`
	HatcheryName             string            `json:"hatchery_name"`
	APIEndpoint              string            `json:"api_endpoint"`
	APIEndpointInsecure      bool              `json:"api_endpoint_insecure,omitempty"`
	APIToken                 string            `json:"api_token"`
	CDNEndpoint              string            `json:"cdn_endpoint"`
	GelfServiceAddr          string            `json:"gelf_service_addr"`
	GelfServiceAddrEnableTLS bool              `json:"gelf_service_addr_enable_tls,omitempty"`
	Model                    string            `json:"model"`
	BookedJobID              int64             `json:"booked_job_id,omitempty"`
	RunJobID                 string            `json:"run_job_id,omitempty"`
	Region                   string            `json:"region,omitempty"`
	InjectEnvVars            map[string]string `json:"inject_env_vars,omitempty"`
}

func (cfg WorkerConfig) EncodeBase64() string {
	btes, _ := json.Marshal(cfg)
	return base64.StdEncoding.EncodeToString(btes)
}

type DownloadArtifact struct {
	Workflow    string `json:"workflow"`
	Number      int64  `json:"number"`
	Pattern     string `json:"pattern" cli:"pattern"`
	Tag         string `json:"tag" cli:"tag"`
	Destination string `json:"destination"`
}

type UploadArtifact struct {
	Name             string `json:"name"`
	Tag              string `json:"tag"`
	WorkingDirectory string `json:"working_directory"`
}

type FilePath struct {
	Path string `json:"path"`
}

type KeyResponse struct {
	PKey    string      `json:"pkey"`
	Type    sdk.KeyType `json:"type"`
	Content []byte      `json:"-"`
}

type TmplPath struct {
	Path        string `json:"path"`
	Destination string `json:"destination"`
}

type CDSVersionSet struct {
	Value string `json:"value"`
}

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

type (
	contextKey int
)

const (
	jobID contextKey = iota
	stepOrder
	stepName
	workDir
	keysDir
	tmpDir
)

type Runtime interface {
	Name() string
	Register(ctx context.Context) error
	Take(ctx context.Context, job sdk.WorkflowNodeJobRun) error
	ProcessJob(job sdk.WorkflowNodeJobRunData) sdk.Result
	SendLog(ctx context.Context, level Level, format string)
	RunResultSignature(fileName string, perm uint32, t sdk.WorkflowRunResultType) (string, error)
	WorkerCacheSignature(tag string) (string, error)
	FeatureEnabled(featureName sdk.FeatureName) bool
	GetIntegrationPlugin(pluginType string) *sdk.GRPCPlugin
	GetActionPlugin(pluginName string) *sdk.GRPCPlugin
	SetActionPlugin(p *sdk.GRPCPlugin)
	GetJobIdentifiers() (int64, int64, int64)
	CDNHttpURL() string
	InstallKey(key sdk.Variable) (*KeyResponse, error)
	InstallKeyTo(key sdk.Variable, destinationPath string) (*KeyResponse, error)
	Unregister(ctx context.Context) error
	Client() cdsclient.WorkerInterface
	BaseDir() afero.Fs
	Environ() []string
	Blur(interface{}) error
	HTTPPort() int32
	Parameters() []sdk.Parameter
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

func StepName(ctx context.Context) (string, error) {
	stepNameInt := ctx.Value(stepName)
	stepName, ok := stepNameInt.(string)
	if !ok {
		return "", fmt.Errorf("unable to get step name: got %v", stepName)
	}
	return stepName, nil
}

func SetStepName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, stepName, name)
}

func WorkingDirectory(ctx context.Context) (afero.File, error) {
	wdi := ctx.Value(workDir)
	wd, ok := wdi.(afero.File)
	if !ok {
		return nil, sdk.WithStack(errors.New("unable to get working directory"))
	}
	log.Debug(ctx, "WorkingDirectory> working directory is : %s", wd.Name())
	return wd, nil
}

func SetWorkingDirectory(ctx context.Context, s afero.File) context.Context {
	log.Debug(ctx, "SetWorkingDirectory> working directory is: %s", s.Name())
	return context.WithValue(ctx, workDir, s)
}

func KeysDirectory(ctx context.Context) (afero.File, error) {
	wdi := ctx.Value(keysDir)
	wd, ok := wdi.(afero.File)
	if !ok {
		return nil, fmt.Errorf("unable to get key directory (%T) %v", wdi, wdi)
	}
	log.Debug(ctx, "KeysDirectory> working directory is : %s", wd.Name())
	return wd, nil
}

func SetKeysDirectory(ctx context.Context, s afero.File) context.Context {
	log.Debug(ctx, "SetKeysDirectory> working directory is: %s", s.Name())
	return context.WithValue(ctx, keysDir, s)
}

func TmpDirectory(ctx context.Context) (afero.File, error) {
	wdi := ctx.Value(tmpDir)
	wd, ok := wdi.(afero.File)
	if !ok {
		return nil, fmt.Errorf("unable to get tmp directory (%T) %v", wdi, wdi)
	}
	log.Debug(ctx, "TmpDirectory> working directory is : %s", wd.Name())
	return wd, nil
}

func SetTmpDirectory(ctx context.Context, s afero.File) context.Context {
	log.Debug(ctx, "SetTmpDirectory> working directory is: %s", s.Name())
	return context.WithValue(ctx, tmpDir, s)
}
