package internal

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"

	"github.com/spf13/afero"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// WorkerServerPort is name of environment variable set to local worker HTTP server port
const WorkerServerPort = "CDS_EXPORT_PORT"

type CurrentWorker struct {
	id         string
	model      sdk.Model
	basedir    afero.Fs
	manualExit bool
	logger     struct {
		logChan chan sdk.Log
		llist   *list.List
	}
	httpPort int32
	register struct {
		apiEndpoint string
		token       string
		model       string
	}
	currentJob struct {
		wJob         *sdk.WorkflowNodeJobRun
		newVariables []sdk.Variable
		gitsshPath   string
		params       []sdk.Parameter
		secrets      []sdk.Variable
		context      context.Context
	}
	status struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	client cdsclient.WorkerInterface
}

// BuiltInAction defines builtin action signature
type BuiltInAction func(context.Context, workerruntime.Runtime, sdk.Action, []sdk.Parameter, []sdk.Variable) (sdk.Result, error)

func (wk *CurrentWorker) Init(name, hatcheryName, apiEndpoint, token string, model string, insecure bool, workspace afero.Fs) error {
	wk.status.Name = name
	wk.basedir = workspace
	wk.register.model = model
	wk.register.token = token
	wk.register.apiEndpoint = apiEndpoint
	wk.client = cdsclient.NewWorker(apiEndpoint, name, cdsclient.NewHTTPClient(time.Second*360, insecure))
	return nil
}

func (wk *CurrentWorker) SendLog(ctx context.Context, level workerruntime.Level, s string) {
	jobID, _ := workerruntime.JobID(ctx)
	stepOrder, err := workerruntime.StepOrder(ctx)
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	if err != nil {
		log.Error("SendLog> %v", err)
	}
	wk.sendLog(jobID, fmt.Sprintf("[%s] ", level)+s, stepOrder, false)
}

func (wk *CurrentWorker) Name() string {
	return wk.status.Name
}

func (wk *CurrentWorker) Client() cdsclient.WorkerInterface {
	return wk.client
}

func (wk *CurrentWorker) Workspace() afero.Fs {
	return wk.basedir
}

func (w *CurrentWorker) Environ() []string {
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
	newEnv = append(newEnv, fmt.Sprintf("%s=%d", WorkerServerPort, w.HTTPPort()))

	//set up environment variables from pipeline build job parameters
	for _, p := range w.currentJob.params {
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

	for _, p := range w.currentJob.newVariables {
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

	if err := json.Unmarshal([]byte(dataS), i); err != nil {
		return err
	}

	return nil
}

func (w *CurrentWorker) HTTPPort() int32 {
	return w.httpPort
}
