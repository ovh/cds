package internal

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"gopkg.in/square/go-jose.v2"

	"github.com/ovh/cds/engine/worker/internal/plugin"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
	loghook "github.com/ovh/cds/sdk/log/hook"
)

const (
	// WorkerServerPort is name of environment variable set to local worker HTTP server port
	WorkerServerPort = "CDS_EXPORT_PORT"

	// CDS API URL
	CDSApiUrl = "CDS_API_URL"
	CDSCDNUrl = "CDS_CDN_URL"
)

type logger struct {
	hook   *loghook.Hook
	logger *logrus.Logger
}

type CurrentJobV2 struct {
	runJob           *sdk.V2WorkflowRunJob
	runJobContext    sdk.WorkflowRunJobsContext
	context          context.Context
	currentStepIndex int
	currentStepName  string
	integrations     map[string]sdk.ProjectIntegration // contains integration with clearPassword
	envFromHooks     map[string]string
	sensitiveDatas   []string
}

type CurrentWorker struct {
	cfg           *workerruntime.WorkerConfig
	id            string
	model         sdk.Model
	basedir       afero.Fs
	workingDirAbs string
	manualExit    bool
	gelfLogger    *logger
	stepLogLine   int64
	httpPort      int32
	signer        jose.Signer
	actionPlugin  map[string]*sdk.GRPCPlugin
	actions       map[string]sdk.V2Action
	pluginFactory plugin.Factory
	currentJobV2  CurrentJobV2
	currentJob    struct {
		wJob             *sdk.WorkflowNodeJobRun
		newVariables     []sdk.Variable
		params           []sdk.Parameter
		secrets          []sdk.Variable
		context          context.Context
		projectKey       string
		workflowName     string
		workflowID       int64
		runID            int64
		runNumber        int64
		nodeRunName      string
		features         map[sdk.FeatureName]bool
		currentStepIndex int
		currentStepName  string
		envFromHooks     map[string]string
	}
	status struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	clientV2 cdsclient.V2WorkerInterface
	client   cdsclient.WorkerInterface
	blur     *sdk.Blur
	hooks    []workerHook
}

type workerHook struct {
	Config       sdk.WorkerHookSetupTeardownScripts
	SetupPath    string
	TeardownPath string
}

// BuiltInAction defines builtin action signature
type BuiltInAction func(context.Context, workerruntime.Runtime, sdk.Action, []sdk.Variable) (sdk.Result, error)

func (wk *CurrentWorker) Init(cfg *workerruntime.WorkerConfig, workspace afero.Fs) error {
	wk.cfg = cfg
	wk.status.Name = cfg.Name
	wk.basedir = workspace
	if sdk.IsValidUUID(cfg.RunJobID) {
		wk.clientV2 = cdsclient.NewWorkerV2(cfg.APIEndpoint, cfg.Name, cdsclient.NewHTTPClient(time.Second*30, cfg.APIEndpointInsecure))
	} else {
		wk.client = cdsclient.NewWorker(cfg.APIEndpoint, cfg.Name, cdsclient.NewHTTPClient(time.Second*30, cfg.APIEndpointInsecure))
	}
	wk.pluginFactory = &plugin.PluginFactory{}
	return nil
}

func (wk *CurrentWorker) GetJobIdentifiers() (int64, int64, int64) {
	return wk.currentJob.runID, wk.currentJob.wJob.WorkflowNodeRunID, wk.currentJob.wJob.ID
}

func (wk *CurrentWorker) GetContext() context.Context {
	if wk.currentJob.context != nil {
		return wk.currentJob.context
	}
	return wk.currentJobV2.context
}

// used into unit tests only
func (wk *CurrentWorker) SetContextForTestJobV2(t *testing.T, c context.Context) {
	wk.currentJobV2.context = c
}

func (wk *CurrentWorker) SetGelfLogger(h *loghook.Hook, l *logrus.Logger) {
	wk.gelfLogger = new(logger)
	wk.gelfLogger.logger = l
	wk.gelfLogger.hook = h
}

func (wk *CurrentWorker) Parameters() []sdk.Parameter {
	return wk.currentJob.params
}

func (wk *CurrentWorker) FeatureEnabled(name sdk.FeatureName) bool {
	b, has := wk.currentJob.features[name]
	if !has {
		return false
	}
	return b
}

func (wk *CurrentWorker) SendTerminatedStepLog(ctx context.Context, level workerruntime.Level, logLine string) {
	msg, sign, err := wk.prepareLog(ctx, level, logLine)
	if err != nil {
		log.Error(wk.GetContext(), "unable to prepare log: %v", err)
		return
	}
	wk.gelfLogger.logger.
		WithField(cdslog.ExtraFieldSignature, sign).
		WithField(cdslog.ExtraFieldLine, wk.stepLogLine).
		WithField(cdslog.ExtraFieldTerminated, true).
		Log(msg.Level, msg.Value)
	wk.stepLogLine++
}

func (wk *CurrentWorker) WorkerCacheSignature(tag string) (string, error) {
	sig := cdn.Signature{
		ProjectKey: wk.currentJob.projectKey,
		Worker: &cdn.SignatureWorker{
			WorkerID:   wk.id,
			WorkerName: wk.Name(),
			CacheTag:   tag,
		},
	}
	signature, err := jws.Sign(wk.signer, sig)
	return signature, sdk.WrapError(err, "cannot sign log message")
}

func (wk *CurrentWorker) GetActionPlugin(pluginName string) *sdk.GRPCPlugin {
	return wk.actionPlugin[pluginName]
}
func (wk *CurrentWorker) SetActionPlugin(p *sdk.GRPCPlugin) {
	wk.actionPlugin[p.Name] = p
}

func (wk *CurrentWorker) GetIntegrationPlugin(pluginType string) *sdk.GRPCPlugin {
	for i := range wk.currentJob.wJob.IntegrationPlugins {
		if wk.currentJob.wJob.IntegrationPlugins[i].Type == pluginType {
			return &wk.currentJob.wJob.IntegrationPlugins[i]
		}
	}
	return nil
}

func (wk *CurrentWorker) RunResultSignature(artifactName string, perm uint32, t sdk.WorkflowRunResultType) (string, error) {
	sig := cdn.Signature{
		ProjectKey:   wk.currentJob.projectKey,
		JobID:        wk.currentJob.wJob.ID,
		NodeRunID:    wk.currentJob.wJob.WorkflowNodeRunID,
		Timestamp:    time.Now().UnixNano(),
		WorkflowID:   wk.currentJob.workflowID,
		WorkflowName: wk.currentJob.workflowName,
		NodeRunName:  wk.currentJob.nodeRunName,
		RunID:        wk.currentJob.runID,
		RunNumber:    wk.currentJob.runNumber,
		JobName:      wk.currentJob.wJob.Job.Action.Name,
		Worker: &cdn.SignatureWorker{
			WorkerID:      wk.id,
			WorkerName:    wk.Name(),
			FileName:      artifactName,
			FilePerm:      perm,
			RunResultType: string(t),
		},
	}
	signature, err := jws.Sign(wk.signer, sig)
	return signature, sdk.WrapError(err, "cannot sign log message")
}

func (wk *CurrentWorker) SendLog(ctx context.Context, level workerruntime.Level, logLine string) {
	msg, sign, err := wk.prepareLog(ctx, level, logLine)
	if err != nil {
		log.Error(wk.GetContext(), "unable to prepare log: %v", err)
		return
	}

	if isReadinessServices, _ := workerruntime.IsReadinessServices(ctx); isReadinessServices {
		log.Info(ctx, msg.Value)
		return
	}

	wk.gelfLogger.logger.
		WithField(cdslog.ExtraFieldSignature, sign).
		WithField(cdslog.ExtraFieldLine, wk.stepLogLine).
		WithField(cdslog.ExtraFieldTerminated, false).
		Log(msg.Level, msg.Value)
	wk.stepLogLine++
}

func (wk *CurrentWorker) CDNHttpURL() string {
	return wk.cfg.CDNEndpoint
}

func (wk *CurrentWorker) prepareLog(ctx context.Context, level workerruntime.Level, s string) (cdslog.Message, string, error) {
	var ts = time.Now().UnixNano()
	var res cdslog.Message

	if wk.blur == nil {
		return res, "", sdk.WithStack(fmt.Errorf("blur is nil"))
	}
	s = wk.blur.String(s)

	switch level {
	case workerruntime.LevelDebug:
		res.Level = logrus.DebugLevel
	case workerruntime.LevelInfo:
		res.Level = logrus.InfoLevel
	case workerruntime.LevelWarn:
		res.Level = logrus.WarnLevel
	case workerruntime.LevelError:
		res.Level = logrus.ErrorLevel
	}

	stepOrder, _ := workerruntime.StepOrder(ctx)
	stepName, _ := workerruntime.StepName(ctx)

	switch {
	case wk.currentJob.wJob != nil:
		res.Signature = cdn.Signature{
			Worker: &cdn.SignatureWorker{
				WorkerID:   wk.id,
				WorkerName: wk.Name(),
				StepOrder:  int64(stepOrder),
				StepName:   stepName,
			},
			ProjectKey:   wk.currentJob.projectKey,
			JobID:        wk.currentJob.wJob.ID,
			NodeRunID:    wk.currentJob.wJob.WorkflowNodeRunID,
			Timestamp:    ts,
			WorkflowID:   wk.currentJob.workflowID,
			WorkflowName: wk.currentJob.workflowName,
			NodeRunName:  wk.currentJob.nodeRunName,
			RunID:        wk.currentJob.runID,
			RunNumber:    wk.currentJob.runNumber,
			JobName:      wk.currentJob.wJob.Job.Action.Name,
		}
	case wk.currentJobV2.runJob != nil:
		res.Signature = cdn.Signature{
			Worker: &cdn.SignatureWorker{
				WorkerID:   wk.id,
				WorkerName: wk.Name(),
				StepOrder:  int64(stepOrder),
				StepName:   stepName,
			},
			ProjectKey:    wk.currentJobV2.runJob.ProjectKey,
			RunJobID:      wk.currentJobV2.runJob.ID,
			Timestamp:     ts,
			WorkflowName:  wk.currentJobV2.runJob.WorkflowName,
			WorkflowRunID: wk.currentJobV2.runJob.WorkflowRunID,
			RunNumber:     wk.currentJobV2.runJob.RunNumber,
			RunAttempt:    wk.currentJobV2.runJob.RunAttempt,
			JobName:       wk.currentJobV2.runJob.JobID,
			Region:        wk.currentJobV2.runJob.Region,
		}
	default:
		return res, "", sdk.WithStack(fmt.Errorf("job is nil"))
	}

	res.Value = s
	signature, err := jws.Sign(wk.signer, res.Signature)
	if err != nil {
		return res, "", sdk.WrapError(err, "cannot sign log message")
	}

	return res, signature, nil
}

func (wk *CurrentWorker) Name() string {
	return wk.status.Name
}

func (wk *CurrentWorker) ClientV2() cdsclient.V2WorkerInterface {
	return wk.clientV2
}

func (wk *CurrentWorker) Client() cdsclient.WorkerInterface {
	return wk.client
}

func (wk *CurrentWorker) BaseDir() afero.Fs {
	return wk.basedir
}

func (wk *CurrentWorker) PluginGetBinary(name, os, arch string, w io.Writer) error {
	if wk.Client() != nil {
		return wk.Client().PluginGetBinary(name, os, arch, w)
	}
	return wk.ClientV2().PluginGetBinary(name, os, arch, w)
}

func (wk *CurrentWorker) PluginGet(pluginName string) (*sdk.GRPCPlugin, error) {
	if wk.Client() != nil {
		return wk.Client().PluginsGet(pluginName)
	}
	return wk.ClientV2().PluginsGet(pluginName)
}

func (wk *CurrentWorker) Environ() []string {
	env := os.Environ()
	newEnv := []string{"CI=1"}
	if sdk.IsValidUUID(wk.cfg.RunJobID) {
		newEnv = append(newEnv, "CDS_LEGACY_MODE=false")
	}

	// filter technical env variables
	for _, e := range env {
		if e == "" {
			continue
		}
		if strings.HasPrefix(e, "CDS_") {
			continue
		}
		newEnv = append(newEnv, e)
	}

	newEnv = append(newEnv, "CDS_KEY=********") //We have to let it here for some legacy reason
	newEnv = append(newEnv, fmt.Sprintf("%s=%d", WorkerServerPort, wk.HTTPPort()))
	newEnv = append(newEnv, fmt.Sprintf("%s=%s", CDSApiUrl, wk.cfg.APIEndpoint))
	newEnv = append(newEnv, fmt.Sprintf("%s=%s", CDSCDNUrl, wk.cfg.CDNEndpoint))

	newEnv = append(newEnv, "HATCHERY_NAME="+wk.cfg.HatcheryName)
	newEnv = append(newEnv, "HATCHERY_WORKER="+wk.cfg.Name)
	if wk.cfg.Region != "" {
		newEnv = append(newEnv, "HATCHERY_REGION="+wk.cfg.Region)
	}
	if wk.cfg.Model != "" {
		newEnv = append(newEnv, "HATCHERY_MODEL="+wk.cfg.Model)
	}
	for k, v := range wk.cfg.InjectEnvVars {
		if v == "" {
			continue
		}
		newEnv = append(newEnv, k+"="+sdk.OneLineValue(v))
	}

	if !sdk.IsValidUUID(wk.cfg.RunJobID) {
		if wk.currentJob.wJob != nil {
			data := []byte(wk.currentJob.wJob.Job.Job.Action.Name)
			suffix := fmt.Sprintf("%x", md5.Sum(data))
			newEnv = append(newEnv, "BASEDIR="+wk.cfg.Basedir+"/"+suffix)
		} else {
			newEnv = append(newEnv, "BASEDIR="+wk.cfg.Basedir)
		}

		//set up environment variables from pipeline build job parameters
		for _, p := range wk.currentJob.params {
			// avoid put private key in environment var as it's a binary value
			if strings.HasPrefix(p.Name, "cds.key.") && strings.HasSuffix(p.Name, ".priv") {
				continue
			}

			newEnv = append(newEnv, sdk.EnvVartoENV(p)...)

			envName := strings.Replace(p.Name, ".", "_", -1)
			envName = strings.Replace(envName, "-", "_", -1)
			envName = strings.ToUpper(envName)
			newEnv = append(newEnv, fmt.Sprintf("%s=%s", envName, sdk.OneLineValue(p.Value)))
		}

		for _, p := range wk.currentJob.newVariables {
			envName := strings.Replace(p.Name, ".", "_", -1)
			envName = strings.Replace(envName, "-", "_", -1)
			envName = strings.ToUpper(envName)
			newEnv = append(newEnv, fmt.Sprintf("%s=%s", envName, sdk.OneLineValue(p.Value)))
		}

		//Set env variables from hooks
		for k, v := range wk.currentJob.envFromHooks {
			newEnv = append(newEnv, k+"="+sdk.OneLineValue(v))
		}
	} else {
		data := []byte(wk.currentJobV2.runJob.JobID)
		suffix := fmt.Sprintf("%x", md5.Sum(data))
		newEnv = append(newEnv, "BASEDIR="+wk.cfg.Basedir+"/"+suffix)
		for k, v := range wk.currentJobV2.envFromHooks {
			newEnv = append(newEnv, k+"="+sdk.OneLineValue(v))
		}
	}
	return newEnv
}

func (w *CurrentWorker) Blur(i interface{}) error {
	if w.blur == nil {
		return fmt.Errorf("blur is not define for current worker")
	}

	return w.blur.Interface(i)
}

func (w *CurrentWorker) HTTPPort() int32 {
	return w.httpPort
}

func (wk *CurrentWorker) SetSecrets(secrets []sdk.Variable) error {
	wk.currentJob.secrets = secrets

	values := make([]string, len(wk.currentJob.secrets))
	for i := range wk.currentJob.secrets {
		values[i] = wk.currentJob.secrets[i].Value
	}

	b, err := sdk.NewBlur(values)
	if err != nil {
		return err
	}

	wk.blur = b

	return nil
}
