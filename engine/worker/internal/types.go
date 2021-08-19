package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"gopkg.in/square/go-jose.v2"

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
)

type logger struct {
	hook   *loghook.Hook
	logger *logrus.Logger
}

type CurrentWorker struct {
	id          string
	model       sdk.Model
	basedir     afero.Fs
	manualExit  bool
	gelfLogger  *logger
	cdnHttpAddr string
	stepLogLine int64
	httpPort    int32
	register    struct {
		apiEndpoint string
		token       string
		model       string
	}
	currentJob struct {
		wJob             *sdk.WorkflowNodeJobRun
		newVariables     []sdk.Variable
		params           []sdk.Parameter
		secrets          []sdk.Variable
		context          context.Context
		signer           jose.Signer
		projectKey       string
		workflowName     string
		workflowID       int64
		runID            int64
		runNumber        int64
		nodeRunName      string
		features         map[sdk.FeatureName]bool
		currentStepIndex int
		currentStepName  string
	}
	status struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	client cdsclient.WorkerInterface
}

// BuiltInAction defines builtin action signature
type BuiltInAction func(context.Context, workerruntime.Runtime, sdk.Action, []sdk.Variable) (sdk.Result, error)

func (wk *CurrentWorker) Init(name, hatcheryName, apiEndpoint, token string, model string, insecure bool, workspace afero.Fs) error {
	wk.status.Name = name
	wk.basedir = workspace
	wk.register.model = model
	wk.register.token = token
	wk.register.apiEndpoint = apiEndpoint
	wk.client = cdsclient.NewWorker(apiEndpoint, name, cdsclient.NewHTTPClient(time.Second*10, insecure))
	return nil
}

func (wk *CurrentWorker) GetJobIdentifiers() (int64, int64, int64) {
	return wk.currentJob.runID, wk.currentJob.wJob.WorkflowNodeRunID, wk.currentJob.wJob.ID
}

func (wk *CurrentWorker) GetContext() context.Context {
	return wk.currentJob.context
}

func (wk *CurrentWorker) SetContext(c context.Context) {
	wk.currentJob.context = c
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
	signature, err := jws.Sign(wk.currentJob.signer, sig)
	return signature, sdk.WrapError(err, "cannot sign log message")
}

func (wk *CurrentWorker) GetPlugin(pluginType string) *sdk.GRPCPlugin {
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
	signature, err := jws.Sign(wk.currentJob.signer, sig)
	return signature, sdk.WrapError(err, "cannot sign log message")
}

func (wk *CurrentWorker) SendLog(ctx context.Context, level workerruntime.Level, logLine string) {
	msg, sign, err := wk.prepareLog(ctx, level, logLine)
	if err != nil {
		log.Error(wk.GetContext(), "unable to prepare log: %v", err)
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
	return wk.cdnHttpAddr
}

func (wk *CurrentWorker) prepareLog(ctx context.Context, level workerruntime.Level, s string) (cdslog.Message, string, error) {
	var res cdslog.Message

	if wk.currentJob.wJob == nil {
		return res, "", sdk.WithStack(fmt.Errorf("job is nill"))
	}
	if err := wk.Blur(&s); err != nil {
		return res, "", sdk.WrapError(err, "unable to blur log")
	}

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
		Timestamp:    time.Now().UnixNano(),
		WorkflowID:   wk.currentJob.workflowID,
		WorkflowName: wk.currentJob.workflowName,
		NodeRunName:  wk.currentJob.nodeRunName,
		RunID:        wk.currentJob.runID,
		RunNumber:    wk.currentJob.runNumber,
		JobName:      wk.currentJob.wJob.Job.Action.Name,
	}

	res.Value = s

	signature, err := jws.Sign(wk.currentJob.signer, res.Signature)
	if err != nil {
		return res, "", sdk.WrapError(err, "cannot sign log message")
	}

	return res, signature, nil
}

func (wk *CurrentWorker) Name() string {
	return wk.status.Name
}

func (wk *CurrentWorker) Client() cdsclient.WorkerInterface {
	return wk.client
}

func (wk *CurrentWorker) BaseDir() afero.Fs {
	return wk.basedir
}

func (wk *CurrentWorker) Environ() []string {
	env := os.Environ()
	newEnv := []string{"CI=1"}
	// filter technical env variables
	for _, e := range env {
		if strings.HasPrefix(e, "CDS_") {
			continue
		}
		newEnv = append(newEnv, e)
	}

	//We have to let it here for some legacy reason
	newEnv = append(newEnv, "CDS_KEY=********")

	// worker export http port
	newEnv = append(newEnv, fmt.Sprintf("%s=%d", WorkerServerPort, wk.HTTPPort()))

	// Api Endpoint in CDS_API_URL var
	newEnv = append(newEnv, fmt.Sprintf("%s=%s", CDSApiUrl, wk.register.apiEndpoint))

	//set up environment variables from pipeline build job parameters
	for _, p := range wk.currentJob.params {
		// avoid put private key in environment var as it's a binary value
		if strings.HasPrefix(p.Name, "cds.key.") && strings.HasSuffix(p.Name, ".priv") {
			continue
		}
		if p.Type == sdk.KeyParameter && !strings.HasSuffix(p.Name, ".pub") {
			continue
		}

		newEnv = append(newEnv, sdk.EnvVartoENV(p)...)

		envName := strings.Replace(p.Name, ".", "_", -1)
		envName = strings.Replace(envName, "-", "_", -1)
		envName = strings.ToUpper(envName)
		newEnv = append(newEnv, fmt.Sprintf("%s=%s", envName, p.Value))
	}

	for _, p := range wk.currentJob.newVariables {
		envName := strings.Replace(p.Name, ".", "_", -1)
		envName = strings.Replace(envName, "-", "_", -1)
		envName = strings.ToUpper(envName)
		newEnv = append(newEnv, fmt.Sprintf("%s=%s", envName, p.Value))
	}
	return newEnv
}

func (w *CurrentWorker) Blur(i interface{}) error {
	data, err := json.Marshal(i)
	if err != nil {
		return err
	}

	dataS := string(data)
	for i := range w.currentJob.secrets {
		if len(w.currentJob.secrets[i].Value) >= sdk.SecretMinLength {
			dataS = strings.Replace(dataS, w.currentJob.secrets[i].Value, sdk.PasswordPlaceholder, -1)
		}
	}

	if err := sdk.JSONUnmarshal([]byte(dataS), i); err != nil {
		return err
	}

	return nil
}

func (w *CurrentWorker) HTTPPort() int32 {
	return w.httpPort
}
