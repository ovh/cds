package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/cds/sdk/log"
)

func processJobParameter(params []sdk.Parameter, secrets []sdk.Variable) {
	parameters := params

	for i := range parameters {
		keepReplacing := true
		for keepReplacing {
			t := parameters[i].Value

			for _, p := range parameters {
				parameters[i].Value = strings.Replace(parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
			}

			for _, p := range secrets {
				parameters[i].Value = strings.Replace(parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
			}

			// If parameters wasn't updated, consider it done
			if parameters[i].Value == t {
				keepReplacing = false
			}
		}
	}

	params = parameters
	return
}

// ProcessActionVariables replaces all placeholders inside action recursively using
// - parent parameters
// - action build arguments
// - Secrets from project, application and environment
//
// This function should be called ONLY from worker
func (w *CurrentWorker) processActionVariables(a *sdk.Action, parent *sdk.Action, jobParameters []sdk.Parameter, secrets []sdk.Variable) error {
	// replaces placeholder in parameters with ActionBuild variables
	// replaces placeholder in parameters with Parent params
	for i := range a.Parameters {
		keepReplacing := true
		for keepReplacing {
			t := a.Parameters[i].Value

			if parent != nil {
				for _, p := range parent.Parameters {
					a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
				}
			}

			for _, p := range jobParameters {
				a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
			}

			for _, p := range secrets {
				a.Parameters[i].Value = strings.Replace(a.Parameters[i].Value, "{{."+p.Name+"}}", p.Value, -1)
			}

			// If parameters wasn't updated, consider it done
			if a.Parameters[i].Value == t {
				keepReplacing = false
			}
		}
	}

	// replaces placeholder in all children recursively
	for i := range a.Actions {
		if err := w.processActionVariables(&a.Actions[i], a, jobParameters, secrets); err != nil {
			return nil
		}
	}

	return nil
}

func (w *CurrentWorker) replaceVariablesPlaceholder(a *sdk.Action, params []sdk.Parameter) error {
	tmp := sdk.ParametersToMap(params)
	for i := range a.Parameters {
		var err error
		a.Parameters[i].Value, err = interpolate.Do(a.Parameters[i].Value, tmp)
		if err != nil {
			return sdk.WrapError(err, "Unable to interpolate action parameters")
		}
	}
	return nil
}

func (w *CurrentWorker) runJob(ctx context.Context, a *sdk.Action, jobID int64, params []sdk.Parameter, secrets []sdk.Variable) (sdk.Result, error) {
	log.Info("runJob> start job %s (%d)", a.Name, jobID)
	defer func() { log.Info("runJob> job %s (%d)", a.Name, jobID) }()

	var jobResult = sdk.Result{
		Status:  sdk.StatusSuccess,
		BuildID: jobID,
	}

	var nDisabled, nCriticalFailed int
	for jobStepIndex, step := range a.Actions {
		ctx = workerruntime.SetStepOrder(ctx, jobStepIndex)
		if err := w.updateStepStatus(ctx, jobID, jobStepIndex, sdk.StatusBuilding); err != nil {
			jobResult.Status = sdk.StatusFail
			jobResult.Reason = fmt.Sprintf("Cannot update step (%d) status (%s): %v", jobStepIndex, sdk.StatusBuilding, err)
			return jobResult, err
		}
		var stepResult = sdk.Result{
			Status:  sdk.StatusNeverBuilt,
			BuildID: jobID,
		}
		if nCriticalFailed == 0 || step.AlwaysExecuted {
			stepResult = w.runAction(ctx, step, jobID, params, secrets, step.Name)
			// TODO: manage new variables
			switch stepResult.Status {
			case sdk.StatusDisabled:
				nDisabled++
			case sdk.StatusFail:
				if !step.Optional {
					nCriticalFailed++
				}
			}
		}
		if err := w.updateStepStatus(ctx, jobID, jobStepIndex, stepResult.Status); err != nil {
			jobResult.Status = sdk.StatusFail
			jobResult.Reason = fmt.Sprintf("Cannot update step (%d) status (%s): %v", jobStepIndex, sdk.StatusBuilding, err)
			return jobResult, err
		}
	}

	//If all steps are disabled, set action status to disabled
	jobResult.Status = sdk.StatusSuccess
	if nDisabled >= len(a.Actions) {
		jobResult.Status = sdk.StatusDisabled
	}
	if nCriticalFailed > 0 {
		jobResult.Status = sdk.StatusFail
	}
	return jobResult, nil
}

func (w *CurrentWorker) runAction(ctx context.Context, a sdk.Action, jobID int64, params []sdk.Parameter, secrets []sdk.Variable, actionName string) sdk.Result {
	log.Info("runAction> start action %s %d", actionName, jobID)
	defer func() { log.Info("runAction> end action %s run %d", actionName, jobID) }()

	w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("Starting step \"%s\"", actionName))
	var t0 = time.Now()
	defer func() {
		w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("End of step \"%s\" (%s)", actionName, sdk.Round(time.Since(t0), time.Second).String()))
	}()

	//If the action is disabled; skip it
	if !a.Enabled || w.manualExit {
		return sdk.Result{
			Status:  sdk.StatusDisabled,
			BuildID: jobID,
		}
	}

	// Replace variable placeholder that may have been added by last step
	if err := w.replaceVariablesPlaceholder(&a, params); err != nil {
		return sdk.Result{
			Status:  sdk.StatusFail,
			BuildID: jobID,
			Reason:  err.Error(),
		}
	}

	// ExpandEnv over all action parameters, avoid expending "CDS_*" env variables
	if a.Name != sdk.ScriptAction {
		var getFilteredEnv = func(s string) string {
			if strings.HasPrefix(s, "CDS_") {
				return s
			}
			return os.Getenv(s)
		}
		for i := range a.Parameters {
			a.Parameters[i].Value = os.Expand(a.Parameters[i].Value, getFilteredEnv)
		}
	}

	//If the action if a edge of the action tree; run it
	switch a.Type {
	case sdk.BuiltinAction:
		return w.runBuiltin(ctx, a, params, secrets)
	case sdk.PluginAction:
		//Run the plugin
		return w.runGRPCPlugin(ctx, a, params)
	}

	// There is is no children actions (action is empty) to do, success !
	if len(a.Actions) == 0 {
		return sdk.Result{
			Status:  sdk.StatusSuccess,
			BuildID: jobID,
		}
	}

	//Run children actions
	r, nDisabled := w.runSteps(ctx, a.Actions, a, jobID, params, secrets, actionName)
	//If all steps are disabled, set action status to disabled
	if nDisabled >= len(a.Actions) {
		r.Status = sdk.StatusDisabled
	}

	return r
}

func (w *CurrentWorker) runSteps(ctx context.Context, steps []sdk.Action, a sdk.Action, jobID int64, params []sdk.Parameter, secrets []sdk.Variable, stepName string) (sdk.Result, int) {
	log.Info("runSteps> start action steps %s %d len(steps):%d context=%p", stepName, jobID, len(steps), ctx)
	defer func() {
		log.Info("runSteps> end action steps %s %d len(steps):%d context=%p (%s)", stepName, jobID, len(steps), ctx, ctx.Err())
	}()
	var criticalStepFailed bool
	var nbDisabledChildren int

	r := sdk.Result{
		Status:  sdk.StatusFail,
		BuildID: jobID,
	}

	for i, child := range steps {
		childName := fmt.Sprintf("%s/%s-%d", stepName, child.Name, i+1)
		if child.StepName != "" {
			childName = "/" + child.StepName
		}
		if !child.Enabled || w.manualExit {
			nbDisabledChildren++
			continue
		}

		if !criticalStepFailed || child.AlwaysExecuted {
			r = w.runAction(ctx, child, jobID, params, secrets, childName)
			if r.Status != sdk.StatusSuccess && !child.Optional {
				criticalStepFailed = true
			}
		} else if criticalStepFailed && !child.AlwaysExecuted {
			r.Status = sdk.StatusNeverBuilt
		}
	}

	if criticalStepFailed {
		r.Status = sdk.StatusFail
	} else {
		r.Status = sdk.StatusSuccess
	}

	return r, nbDisabledChildren
}

func (w *CurrentWorker) updateStepStatus(ctx context.Context, buildID int64, stepOrder int, status string) error {
	step := sdk.StepStatus{
		StepOrder: stepOrder,
		Status:    status,
		Start:     time.Now(),
		Done:      time.Now(),
	}

	for try := 1; try <= 10; try++ {
		ctxt, cancel := context.WithTimeout(ctx, 120*time.Second)
		lasterr := w.Client().QueueSendStepResult(ctxt, buildID, step)
		if lasterr == nil {
			log.Info("updateStepStatus> Sending step status %s buildID:%d stepOrder:%d", status, buildID, stepOrder)
			cancel()
			return nil
		}
		cancel()
		if ctx.Err() != nil {
			return fmt.Errorf("updateStepStatus> step:%d job:%d worker is cancelled", stepOrder, buildID)
		}
		log.Warning("updateStepStatus> Cannot send step %d result: err: %s - try: %d - new try in 15s", stepOrder, lasterr, try)
		time.Sleep(15 * time.Second)
	}
	return fmt.Errorf("updateStepStatus> Could not send built result 10 times on step %d, giving up. job: %d", stepOrder, buildID)
}

// creates a working directory in $HOME/PROJECT/APP/PIP/BN
func setupBuildDirectory(wd string) error {
	if err := os.MkdirAll(wd, 0755); err != nil {
		return err
	}

	if err := os.Chdir(wd); err != nil {
		return err
	}

	var err error
	u, err := user.Current()
	if err != nil {
		log.Error("Error while getting current user %v", err)
	} else if u != nil && u.HomeDir != "" {
		if err := os.Setenv("HOME_CDS_PLUGINS", u.HomeDir); err != nil {
			log.Error("Error while setting home_plugin %v", err)
		}
	}
	return os.Setenv("HOME", wd)
}

// remove the buildDirectory created by setupBuildDirectory
func teardownBuildDirectory(wd string) error {
	return os.RemoveAll(wd)
}

func workingDirectory(fs afero.Fs, jobInfo sdk.WorkflowNodeJobRunData, suffixes ...string) string {
	var encodedName = base64.RawStdEncoding.EncodeToString([]byte(jobInfo.NodeJobRun.Job.Job.Action.Name))
	paths := append([]string{fs.Name(), encodedName}, suffixes...)
	dir := path.Join(paths...)

	if _, err := fs.Stat(dir); os.IsExist(err) {
		log.Info("workingDirectory> cleaning directory %s", dir)
		_ = os.RemoveAll(dir)
	}
	return dir
}

func (w *CurrentWorker) ProcessJob(jobInfo sdk.WorkflowNodeJobRunData) (sdk.Result, error) {
	ctx := w.currentJob.context
	t0 := time.Now()

	// Timeout must be the same as the goroutine which stop jobs in package api/workflow
	ctx, cancel := context.WithTimeout(ctx, 24*time.Hour)
	log.Info("processJob> Process Job %s (%d)", jobInfo.NodeJobRun.Job.Action.Name, jobInfo.NodeJobRun.ID)
	defer func() {
		log.Info("processJob> Process Job Done %s (%d) :%s", jobInfo.NodeJobRun.Job.Action.Name, jobInfo.NodeJobRun.ID, sdk.Round(time.Since(t0), time.Second).String())
	}()
	defer cancel()

	ctx = workerruntime.SetJobID(ctx, jobInfo.NodeJobRun.ID)
	// start logger routine with a large buffer
	w.logger.logChan = make(chan sdk.Log, 100000)
	go w.logProcessor(ctx, jobInfo.NodeJobRun.ID)
	defer w.drainLogsAndCloseLogger(ctx)

	// Setup working directory
	wd := workingDirectory(w.basedir, jobInfo, "run")
	log.Debug("processJob> Setup workspace - mkdir %s", wd)

	if err := setupBuildDirectory(wd); err != nil {
		log.Debug("processJob> setupBuildDirectory error:%s", err)
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: cannot setup working directory: %s", err),
		}, err
	}
	w.currentJob.workingDirectory = wd

	var jobParameters = jobInfo.NodeJobRun.Parameters

	//Add working directory as job parameter
	jobParameters = append(jobParameters, sdk.Parameter{
		Name:  "cds.workspace",
		Type:  sdk.StringParameter,
		Value: wd,
	})

	// add cds.worker on parameters available
	jobParameters = append(jobParameters, sdk.Parameter{
		Name:  "cds.worker",
		Type:  sdk.StringParameter,
		Value: jobInfo.NodeJobRun.Job.WorkerName,
	})

	// REPLACE ALL VARIABLE EVEN SECRETS HERE
	processJobParameter(jobParameters, jobInfo.Secrets)
	if err := w.processActionVariables(&jobInfo.NodeJobRun.Job.Action, nil, jobParameters, jobInfo.Secrets); err != nil {
		log.Warning("processJob> Cannot process action %s parameters: %s", jobInfo.NodeJobRun.Job.Action.Name, err)
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: cannot process action %s parameters", jobInfo.NodeJobRun.Job.Action.Name),
		}, err
	}

	// Add secrets as string or password in ActionBuild.Args
	// So they can be used by plugins
	for _, s := range jobInfo.Secrets {
		p := sdk.Parameter{
			Type:  s.Type,
			Name:  s.Name,
			Value: s.Value,
		}
		jobParameters = append(jobParameters, p)
	}

	// Setup user ssh keys
	keysDirectory = workingDirectory(w.basedir, jobInfo, "keys")
	log.Debug("processJob> Setup user ssh keys - mkdir %s", keysDirectory)
	if err := os.MkdirAll(keysDirectory, 0700); err != nil {
		log.Debug("processJob> call os.MkdirAll error:%s", err)
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: cannot setup workingDirectory (%s)", err),
		}, err
	}

	/* // DEPRECATED - BEGIN
	if err := w.setupSSHKey(jobInfo.Secrets, keysDirectory); err != nil {
		log.Debug("processJob> call w.setupSSHKey error:%s", err)
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: cannot setup ssh key (%s)", err),
		}
	}
	// DEPRECATED - END

	// The right way to go is :
	if err := vcs.SetupSSHKey(jobInfo.Secrets, keysDirectory, nil); err != nil {
		log.Debug("processJob> call vcs.SetupSSHKey error:%s", err)
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: cannot setup vcs ssh key (%s)", err),
		}
	} */

	res, err := w.runJob(ctx, &jobInfo.NodeJobRun.Job.Action, jobInfo.NodeJobRun.ID, jobParameters, jobInfo.Secrets)

	if err := teardownBuildDirectory(wd); err != nil {
		log.Error("Cannot remove build directory: %s", err)
	}
	if err := teardownBuildDirectory(keysDirectory); err != nil {
		log.Error("Cannot remove keys directory: %s", err)
	}
	return res, err
}
