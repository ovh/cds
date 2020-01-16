package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/cds/sdk/log"
)

func processJobParameter(parameters []sdk.Parameter, secrets []sdk.Variable) {
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

func (w *CurrentWorker) runJob(ctx context.Context, a *sdk.Action, jobID int64, secrets []sdk.Variable) (sdk.Result, error) {
	log.Info(ctx, "runJob> start job %s (%d)", a.Name, jobID)
	defer func() { log.Info(ctx, "runJob> job %s (%d)", a.Name, jobID) }()

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
			stepResult = w.runAction(ctx, step, jobID, secrets, step.Name)

			// Check if all newVariables are in currentJob.params
			// variable can be add in w.currentJob.newVariables by worker command export
			for _, newVariableFromHandler := range w.currentJob.newVariables {
				if sdk.ParameterFind(w.currentJob.params, newVariableFromHandler.Name) == nil {
					w.currentJob.params = append(w.currentJob.params, newVariableFromHandler.ToParameter(""))
				}
			}

			for _, newVariable := range stepResult.NewVariables {
				// append the new variable from a step to the following steps
				w.currentJob.params = append(w.currentJob.params, newVariable.ToParameter("cds.build"))
				// Propagate new variables from step result to jobs result
				w.currentJob.newVariables = append(w.currentJob.newVariables, newVariable)
			}

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

	// Propagate new variables from steps to jobs result
	jobResult.NewVariables = w.currentJob.newVariables

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

func (w *CurrentWorker) runAction(ctx context.Context, a sdk.Action, jobID int64, secrets []sdk.Variable, actionName string) sdk.Result {
	log.Info(ctx, "runAction> start action %s %s %d", a.StepName, actionName, jobID)
	defer func() { log.Info(ctx, "runAction> end action %s %s run %d", a.StepName, actionName, jobID) }()

	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step \"%s\"", actionName))
	var t0 = time.Now()
	defer func() {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step \"%s\" (%s)", actionName, sdk.Round(time.Since(t0), time.Second).String()))
	}()

	//If the action is disabled; skip it
	if !a.Enabled || w.manualExit {
		return sdk.Result{
			Status:  sdk.StatusDisabled,
			BuildID: jobID,
		}
	}

	// Replace variable placeholder that may have been added by last step
	if err := w.replaceVariablesPlaceholder(&a, w.currentJob.params); err != nil {
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
		return w.runBuiltin(ctx, a, secrets)
	case sdk.PluginAction:
		//Run the plugin
		return w.runGRPCPlugin(ctx, a)
	}

	// There is is no children actions (action is empty) to do, success !
	if len(a.Actions) == 0 {
		return sdk.Result{
			Status:  sdk.StatusSuccess,
			BuildID: jobID,
		}
	}

	//Run children actions
	r, nDisabled := w.runSteps(ctx, a.Actions, a, jobID, secrets, actionName)
	//If all steps are disabled, set action status to disabled
	if nDisabled >= len(a.Actions) {
		r.Status = sdk.StatusDisabled
	}

	return r
}

func (w *CurrentWorker) runSteps(ctx context.Context, steps []sdk.Action, a sdk.Action, jobID int64, secrets []sdk.Variable, stepName string) (sdk.Result, int) {
	log.Info(ctx, "runSteps> start action steps %s %d len(steps):%d context=%p", stepName, jobID, len(steps), ctx)
	defer func() {
		log.Info(ctx, "runSteps> end action steps %s %d len(steps):%d context=%p (%s)", stepName, jobID, len(steps), ctx, ctx.Err())
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
			r = w.runAction(ctx, child, jobID, secrets, childName)
			if r.Status != sdk.StatusSuccess && !child.Optional {
				criticalStepFailed = true
			}
		} else if criticalStepFailed && !child.AlwaysExecuted {
			r.Status = sdk.StatusNeverBuilt
		}

		// Check if all newVariables are in currentJob.params
		// variable can be add in w.currentJob.newVariables by worker command export
		for _, newVariableFromHandler := range w.currentJob.newVariables {
			if sdk.ParameterFind(w.currentJob.params, newVariableFromHandler.Name) == nil {
				w.currentJob.params = append(w.currentJob.params, newVariableFromHandler.ToParameter(""))
			}
		}

		for _, newVariable := range r.NewVariables {
			// append the new variable from a chile to the following children
			w.currentJob.params = append(w.currentJob.params, newVariable.ToParameter("cds.build"))
			// Propagate new variables from child result to action
			r.NewVariables = append(r.NewVariables, newVariable)
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
			log.Info(ctx, "updateStepStatus> Sending step status %s buildID:%d stepOrder:%d", status, buildID, stepOrder)
			cancel()
			return nil
		}
		cancel()
		if ctx.Err() != nil {
			return fmt.Errorf("updateStepStatus> step:%d job:%d worker is cancelled", stepOrder, buildID)
		}
		log.Warning(ctx, "updateStepStatus> Cannot send step %d result: err: %s - try: %d - new try in 15s", stepOrder, lasterr, try)
		time.Sleep(15 * time.Second)
	}
	return fmt.Errorf("updateStepStatus> Could not send built result 10 times on step %d, giving up. job: %d", stepOrder, buildID)
}

// creates a working directory in $HOME/PROJECT/APP/PIP/BN
func setupWorkingDirectory(ctx context.Context, fs afero.Fs, wd string) (afero.File, error) {
	log.Debug("creating directory %s in Filesystem %s", wd, fs.Name())
	if err := fs.MkdirAll(wd, 0755); err != nil {
		return nil, err
	}

	u, err := user.Current()
	if err != nil {
		log.Error(ctx, "Error while getting current user %v", err)
	} else if u != nil && u.HomeDir != "" {
		if err := os.Setenv("HOME_CDS_PLUGINS", u.HomeDir); err != nil {
			log.Error(ctx, "Error while setting home_plugin %v", err)
		}
	}

	var absWD string
	if x, ok := fs.(*afero.BasePathFs); ok {
		absWD, _ = x.RealPath(wd)
	} else {
		absWD = wd
	}
	if err := os.Setenv("HOME", absWD); err != nil {
		return nil, err
	}

	fi, err := fs.Open(wd)
	if err != nil {
		return nil, err
	}
	return fi, nil
}

func teardownDirectory(fs afero.Fs, dir string) error {
	return fs.RemoveAll(dir)
}

func workingDirectory(ctx context.Context, fs afero.Fs, jobInfo sdk.WorkflowNodeJobRunData, suffixes ...string) (string, error) {
	var encodedName = base64.RawStdEncoding.EncodeToString([]byte(jobInfo.NodeJobRun.Job.Job.Action.Name))
	paths := append([]string{encodedName}, suffixes...)
	dir := path.Join(paths...)

	if _, err := fs.Stat(dir); os.IsExist(err) {
		log.Info(ctx, "cleaning working directory %s", dir)
		_ = fs.RemoveAll(dir)
	}

	if err := fs.MkdirAll(dir, os.FileMode(0700)); err != nil {
		return dir, sdk.WithStack(err)
	}

	log.Debug("defining working directory %s", dir)
	return dir, nil
}

func (w *CurrentWorker) setupWorkingDirectory(ctx context.Context, jobInfo sdk.WorkflowNodeJobRunData) (afero.File, string, error) {
	wd, err := workingDirectory(ctx, w.basedir, jobInfo, "run")
	if err != nil {
		return nil, "", err
	}

	wdFile, err := setupWorkingDirectory(ctx, w.basedir, wd)
	if err != nil {
		log.Debug("processJob> setupWorkingDirectory error:%s", err)
		return nil, "", err
	}

	wdAbs, err := filepath.Abs(wdFile.Name())
	if err != nil {
		log.Debug("processJob> setupWorkingDirectory error:%s", err)
		return nil, "", err
	}

	switch x := w.basedir.(type) {
	case *afero.BasePathFs:
		wdAbs, err = x.RealPath(wdFile.Name())
		if err != nil {
			return nil, "", err
		}

		wdAbs, err = filepath.Abs(wdAbs)
		if err != nil {
			log.Debug("processJob> setupWorkingDirectory error:%s", err)
			return nil, "", err
		}
	}

	return wdFile, wdAbs, nil
}

func (w *CurrentWorker) setupKeysDirectory(ctx context.Context, jobInfo sdk.WorkflowNodeJobRunData) (afero.File, string, error) {
	keysDirectory, err := workingDirectory(ctx, w.basedir, jobInfo, "keys")
	if err != nil {
		return nil, "", err
	}

	fs := w.basedir
	if err := fs.MkdirAll(keysDirectory, 0700); err != nil {
		return nil, "", err
	}

	kdFile, err := w.basedir.Open(keysDirectory)
	if err != nil {
		return nil, "", err
	}

	kdAbs, err := filepath.Abs(kdFile.Name())
	if err != nil {
		return nil, "", err
	}

	switch x := w.basedir.(type) {
	case *afero.BasePathFs:
		kdAbs, err = x.RealPath(kdFile.Name())
		if err != nil {
			return nil, "", err
		}

		kdAbs, err = filepath.Abs(kdAbs)
		if err != nil {
			return nil, "", err
		}
	}

	return kdFile, kdAbs, nil
}

func (w *CurrentWorker) ProcessJob(jobInfo sdk.WorkflowNodeJobRunData) (sdk.Result, error) {
	ctx := w.currentJob.context
	t0 := time.Now()

	// Timeout must be the same as the goroutine which stop jobs in package api/workflow
	ctx, cancel := context.WithTimeout(ctx, 24*time.Hour)
	log.Info(ctx, "processJob> Process Job %s (%d)", jobInfo.NodeJobRun.Job.Action.Name, jobInfo.NodeJobRun.ID)
	defer func() {
		log.Info(ctx, "processJob> Process Job Done %s (%d) :%s", jobInfo.NodeJobRun.Job.Action.Name, jobInfo.NodeJobRun.ID, sdk.Round(time.Since(t0), time.Second).String())
	}()
	defer cancel()

	ctx = workerruntime.SetJobID(ctx, jobInfo.NodeJobRun.ID)
	// start logger routine with a large buffer
	w.logger.logChan = make(chan sdk.Log, 100000)
	go func() {
		if err := w.logProcessor(ctx, jobInfo.NodeJobRun.ID); err != nil {
			log.Error(ctx, "processJob> Logs processor error: %v", err)
		}
	}()
	defer func() {
		if err := w.drainLogsAndCloseLogger(ctx); err != nil {
			log.Error(ctx, "processJob> Drain logs error: %v", err)
		}
	}()

	wdFile, wdAbs, err := w.setupWorkingDirectory(ctx, jobInfo)
	if err != nil {
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: unable to setup workfing directory: %v", err),
		}, err
	}
	ctx = workerruntime.SetWorkingDirectory(ctx, wdFile)
	log.Debug("processJob> Setup workspace - %s", wdFile.Name())

	kdFile, _, err := w.setupKeysDirectory(ctx, jobInfo)
	if err != nil {
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: unable to setup keys directory: %v", err),
		}, err
	}
	ctx = workerruntime.SetKeysDirectory(ctx, kdFile)
	log.Debug("processJob> Setup key directory - %s", kdFile.Name())

	w.currentJob.context = ctx

	var jobParameters = jobInfo.NodeJobRun.Parameters

	//Add working directory as job parameter
	jobParameters = append(jobParameters, sdk.Parameter{
		Name:  "cds.workspace",
		Type:  sdk.StringParameter,
		Value: wdAbs,
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
		log.Warning(ctx, "processJob> Cannot process action %s parameters: %s", jobInfo.NodeJobRun.Job.Action.Name, err)
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

	w.currentJob.params = jobParameters

	res, err := w.runJob(ctx, &jobInfo.NodeJobRun.Job.Action, jobInfo.NodeJobRun.ID, jobInfo.Secrets)

	if len(res.NewVariables) > 0 {
		log.Debug("processJob> new variables: %v", res.NewVariables)
	}

	// Delete working directory
	if err := teardownDirectory(w.basedir, wdFile.Name()); err != nil {
		log.Error(ctx, "Cannot remove build directory: %s", err)
	}
	// Delelete key directory
	if err := teardownDirectory(w.basedir, kdFile.Name()); err != nil {
		log.Error(ctx, "Cannot remove keys directory: %s", err)
	}
	// Delete all plugins
	if err := teardownDirectory(w.basedir, ""); err != nil {
		log.Error(ctx, "Cannot remove basedir content: %s", err)
	}

	return res, err
}
