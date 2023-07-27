package internal

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/cds/sdk/slug"
)

func processJobParameter(parameters []sdk.Parameter) error {
	for i := range parameters {
		var err error
		var oldValue = parameters[i].Value
		var x int
		var keepReplacing = true
		for keepReplacing && x < 10 {
			var paramMap = sdk.ParametersToMap(parameters)
			parameters[i].Value, err = interpolate.Do(parameters[i].Value, paramMap)
			if err != nil {
				return sdk.WrapError(err, "Unable to interpolate job parameters")
			}

			// If parameters wasn't updated, consider it done
			if parameters[i].Value == oldValue {
				keepReplacing = false
			}
			x++
		}
	}
	return nil
}

// ProcessActionVariables replaces all placeholders inside action recursively using
// - parent parameters
// - action build arguments
// - Secrets from project, application and environment
//
// This function should be called ONLY from worker
func processActionVariables(a *sdk.Action, parent *sdk.Action, jobParameters []sdk.Parameter) error {
	// replaces placeholder in parameters with ActionBuild variables
	// replaces placeholder in parameters with Parent params

	var parentParamMap = map[string]string{}
	if parent != nil {
		parentParamMap = sdk.ParametersToMap(parent.Parameters)
	}
	jobParamMap := sdk.ParametersToMap(jobParameters)
	allParams := sdk.ParametersMapMerge(parentParamMap, jobParamMap)
	for i := range a.Parameters {
		var err error
		a.Parameters[i].Value, err = interpolate.Do(a.Parameters[i].Value, allParams)
		if err != nil {
			return sdk.NewErrorFrom(err, "unable to interpolate action parameter %q", a.Parameters[i].Name)
		}
	}

	// replaces placeholder in all children recursively
	for i := range a.Actions {
		// Do not interpolate yet cds.version variable for child because the value can change during job execution
		filterJobParameters := make([]sdk.Parameter, 0, len(jobParameters))
		for i := range jobParameters {
			if jobParameters[i].Name != "cds.version" {
				filterJobParameters = append(filterJobParameters, jobParameters[i])
			}
		}

		if err := processActionVariables(&a.Actions[i], a, filterJobParameters); err != nil {
			return err
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
			return sdk.NewErrorFrom(err, "unable to interpolate action parameter %q", a.Parameters[i].Name)
		}
	}
	return nil
}

func (w *CurrentWorker) runJob(ctx context.Context, a *sdk.Action, jobID int64, secrets []sdk.Variable) sdk.Result {
	log.Info(ctx, "runJob> start job %s (%d)", a.Name, jobID)
	var jobResult = sdk.Result{
		Status:  sdk.StatusSuccess,
		BuildID: jobID,
	}

	defer func() {
		w.gelfLogger.hook.Flush()
		log.Info(ctx, "runJob> end of job %s (%d)", a.Name, jobID)
	}()

	var nDisabled, nCriticalFailed int
	for jobStepIndex, step := range a.Actions {
		// Reset step log line to 0
		w.stepLogLine = 0

		w.currentJob.currentStepIndex = jobStepIndex
		ctx = workerruntime.SetStepOrder(ctx, jobStepIndex)
		if step.StepName != "" {
			w.currentJob.currentStepName = step.StepName
			ctx = workerruntime.SetStepName(ctx, step.StepName)
		} else {
			w.currentJob.currentStepName = step.Name
			ctx = workerruntime.SetStepName(ctx, step.Name)
		}

		if err := w.updateStepStatus(ctx, jobID, jobStepIndex, sdk.StatusBuilding); err != nil {
			jobResult.Status = sdk.StatusFail
			jobResult.Reason = fmt.Sprintf("Cannot update step (%d) status (%s): %v", jobStepIndex, sdk.StatusBuilding, err)
			return jobResult
		}
		var stepResult = sdk.Result{
			Status:  sdk.StatusNeverBuilt,
			BuildID: jobID,
		}
		if nCriticalFailed == 0 || step.AlwaysExecuted {
			stepResult = w.runRootAction(ctx, step, jobID, secrets, step.Name)

			// Check if all newVariables are in currentJob.params
			// variable can be add in w.currentJob.newVariables by worker command export
			for _, newVariableFromHandler := range w.currentJob.newVariables {
				p := sdk.ParameterFind(w.currentJob.params, newVariableFromHandler.Name)
				if p == nil {
					w.currentJob.params = append(w.currentJob.params, newVariableFromHandler.ToParameter(""))
				} else {
					p.Value = newVariableFromHandler.Value
				}
			}

			for _, newVariable := range stepResult.NewVariables {
				// append the new variable from a step to the following steps
				w.currentJob.params = append(w.currentJob.params, newVariable.ToParameter(""))
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
			return jobResult
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
	return jobResult
}

func (w *CurrentWorker) runRootAction(ctx context.Context, a sdk.Action, jobID int64, secrets []sdk.Variable, actionName string) sdk.Result {
	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step %q", actionName))
	defer func() {
		w.SendTerminatedStepLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step %q", actionName))
		w.gelfLogger.hook.Flush()
	}()
	return w.runAction(ctx, a, jobID, secrets, actionName)
}

func (w *CurrentWorker) runSubAction(ctx context.Context, a sdk.Action, jobID int64, secrets []sdk.Variable, actionName string) sdk.Result {
	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting sub step %q", actionName))
	defer func() {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of sub step %q", actionName))
		w.gelfLogger.hook.Flush()
	}()
	return w.runAction(ctx, a, jobID, secrets, actionName)
}

func (w *CurrentWorker) runAction(ctx context.Context, a sdk.Action, jobID int64, secrets []sdk.Variable, actionName string) sdk.Result {
	log.Info(ctx, "runAction> start %s action %s %s %d", a.Type, a.StepName, actionName, jobID)
	defer func() { log.Info(ctx, "runAction> end action %s %s run %d", a.StepName, actionName, jobID) }()

	//If the action is disabled; skip it
	if !a.Enabled || w.manualExit {
		return sdk.Result{
			Status:  sdk.StatusDisabled,
			BuildID: jobID,
		}
	}

	if a.Type != sdk.AsCodeAction {

		// Replace variable placeholder that may have been added by last step
		if err := w.replaceVariablesPlaceholder(&a, w.currentJob.params); err != nil {
			log.ErrorWithStackTrace(ctx, err)
			w.SendLog(ctx, workerruntime.LevelError, err.Error())
			return sdk.Result{
				Status:  sdk.StatusFail,
				BuildID: jobID,
				Reason:  err.Error(),
			}
		}
		if err := processActionVariables(&a, nil, w.currentJob.params); err != nil {
			log.ErrorWithStackTrace(ctx, err)
			w.SendLog(ctx, workerruntime.LevelError, err.Error())
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

		//Set env variables from hooks
		for k, v := range w.currentJob.envFromHooks {
			os.Setenv(k, v)
		}

	}

	//If the action if a edge of the action tree; run it
	switch a.Type {
	case sdk.BuiltinAction:
		res := w.runBuiltin(ctx, a, secrets)
		return res
	case sdk.PluginAction:
		res := w.runGRPCPlugin(ctx, a)
		return res
	}

	// There is no children actions (action is empty) to do, success !
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
			r = w.runSubAction(ctx, child, jobID, secrets, childName)
			if r.Status != sdk.StatusSuccess && !child.Optional {
				criticalStepFailed = true
			}
		} else if criticalStepFailed && !child.AlwaysExecuted {
			r.Status = sdk.StatusNeverBuilt
		}

		// Check if all newVariables are in currentJob.params
		// variable can be add in w.currentJob.newVariables by worker command export
		for _, newVariableFromHandler := range w.currentJob.newVariables {
			p := sdk.ParameterFind(w.currentJob.params, newVariableFromHandler.Name)
			if p == nil {
				w.currentJob.params = append(w.currentJob.params, newVariableFromHandler.ToParameter(""))
			} else {
				p.Value = newVariableFromHandler.Value
			}
		}

		for _, newVariable := range r.NewVariables {
			// append the new variable from a chile to the following children
			w.currentJob.params = append(w.currentJob.params, newVariable.ToParameter(""))
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
		log.Warn(ctx, "updateStepStatus> Cannot send step %d result: err: %s - try: %d - new try in 15s", stepOrder, lasterr, try)
		time.Sleep(15 * time.Second)
	}
	return fmt.Errorf("updateStepStatus> Could not send built result 10 times on step %d, giving up. job: %d", stepOrder, buildID)
}

func (w *CurrentWorker) ProcessJob(jobInfo sdk.WorkflowNodeJobRunData) (res sdk.Result) {
	ctx := w.currentJob.context
	t0 := time.Now()

	// Timeout must be the same as the goroutine which stop jobs in package api/workflow
	ctx, cancel := context.WithTimeout(ctx, 24*time.Hour)
	log.Info(ctx, "Process Job %s (%d)", jobInfo.NodeJobRun.Job.Action.Name, jobInfo.NodeJobRun.ID)
	defer func() {
		log.Info(ctx, "Process Job Done %s (%d) :%s", jobInfo.NodeJobRun.Job.Action.Name, jobInfo.NodeJobRun.ID, sdk.Round(time.Since(t0), time.Second).String())
	}()
	defer cancel()

	ctx = workerruntime.SetJobID(ctx, jobInfo.NodeJobRun.ID)
	ctx = workerruntime.SetStepOrder(ctx, 0)
	defer func() {
		log.Warn(ctx, "Status: %s | Reason: %s", res.Status, res.Reason)
	}()

	wdFile, wdAbs, err := w.setupWorkingDirectory(ctx, jobInfo.NodeJobRun.Job.Job.Action.Name)
	if err != nil {
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: unable to setup workfing directory: %v", err),
		}
	}
	w.workingDirAbs = wdAbs
	ctx = workerruntime.SetWorkingDirectory(ctx, wdFile)
	log.Debug(ctx, "Setup workspace - %s", wdFile.Name())

	kdFile, _, err := w.setupKeysDirectory(ctx, jobInfo.NodeJobRun.Job.Job.Action.Name)
	if err != nil {
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: unable to setup keys directory: %v", err),
		}
	}
	ctx = workerruntime.SetKeysDirectory(ctx, kdFile)
	log.Debug(ctx, "Setup key directory - %s", kdFile.Name())

	tdFile, _, err := w.setupTmpDirectory(ctx, jobInfo.NodeJobRun.Job.Job.Action.Name)
	if err != nil {
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Error: unable to setup tmp directory: %v", err),
		}
	}
	ctx = workerruntime.SetTmpDirectory(ctx, tdFile)
	log.Debug(ctx, "Setup tmp directory - %s", tdFile.Name())

	hdFile, _, err := w.setupHooksDirectory(ctx, jobInfo.NodeJobRun.Job.Job.Action.Name)
	if err != nil {
		return sdk.Result{Status: sdk.StatusFail, Reason: fmt.Sprintf("Error: unable to setup hooks directory: %v", err)}
	}

	// Exec worker hooks
	log.Info(ctx, "Setup hooks directory: %s", hdFile.Name())
	if err := w.setupHooks(ctx, jobInfo, w.basedir, hdFile.Name()); err != nil {
		return sdk.Result{Status: sdk.StatusFail, Reason: fmt.Sprintf("Error: unable to setup hooks: %v", err)}
	}

	w.currentJob.context = ctx
	w.currentJob.params = jobInfo.NodeJobRun.Parameters

	//Add working directory as job parameter
	w.currentJob.params = append(w.currentJob.params, sdk.Parameter{
		Name:  "cds.workspace",
		Type:  sdk.StringParameter,
		Value: wdAbs,
	})

	// add cds.worker on parameters available
	w.currentJob.params = append(w.currentJob.params, sdk.Parameter{
		Name:  "cds.worker",
		Type:  sdk.StringParameter,
		Value: jobInfo.NodeJobRun.Job.WorkerName,
	})

	// Add secrets as string or password in ActionBuild.Args
	// So they can be used by plugins
	for _, s := range jobInfo.Secrets {
		p := sdk.Parameter{Type: s.Type, Name: s.Name, Value: s.Value}
		w.currentJob.params = append(w.currentJob.params, p)
	}

	// REPLACE ALL VARIABLE EVEN SECRETS HERE
	if err := processJobParameter(w.currentJob.params); err != nil {
		return sdk.Result{Status: sdk.StatusFail, Reason: fmt.Sprintf("unable to process job %s: %v", jobInfo.NodeJobRun.Job.Action.Name, err)}
	}

	log.Info(ctx, "Executing hooks setup from directory: %s", hdFile.Name())
	if err := w.executeHooksSetup(ctx, w.basedir, hdFile.Name()); err != nil {
		return sdk.Result{Status: sdk.StatusFail, Reason: fmt.Sprintf("Error: unable to setup hooks: %v", err)}
	}

	res = w.runJob(ctx, &jobInfo.NodeJobRun.Job.Action, jobInfo.NodeJobRun.ID, jobInfo.Secrets)

	if len(res.NewVariables) > 0 {
		log.Debug(ctx, "new variables: %v", res.NewVariables)
	}

	// Teardown worker hooks
	if err := w.executeHooksTeardown(ctx, w.basedir, hdFile.Name()); err != nil {
		log.Error(ctx, "error while executing teardown hook scripts: %v", err)
	}

	// Delete hooks directory
	if err := teardownDirectory(w.basedir, hdFile.Name()); err != nil {
		log.Error(ctx, "Cannot remove hooks directory: %s", err)
	}
	// Delete working directory
	if err := teardownDirectory(w.basedir, wdFile.Name()); err != nil {
		log.Error(ctx, "Cannot remove build directory: %s", err)
	}
	// Delete key directory
	if err := teardownDirectory(w.basedir, kdFile.Name()); err != nil {
		log.Error(ctx, "Cannot remove keys directory: %s", err)
	}
	// Delete tmp directory
	if err := teardownDirectory(w.basedir, tdFile.Name()); err != nil {
		log.Error(ctx, "Cannot remove tmp directory: %s", err)
	}
	// Delete all plugins
	if err := teardownDirectory(w.basedir, ""); err != nil {
		log.Error(ctx, "Cannot remove basedir content: %s", err)
	}
	return res
}

func (w *CurrentWorker) setupHooks(ctx context.Context, jobInfo sdk.WorkflowNodeJobRunData, fs afero.Fs, workingDir string) error {
	log.Debug(ctx, "Setup hooks")
	if err := fs.MkdirAll(path.Join(workingDir, "setup"), os.FileMode(0700)); err != nil {
		return errors.WithStack(err)
	}
	if err := fs.MkdirAll(path.Join(workingDir, "teardown"), os.FileMode(0700)); err != nil {
		return errors.WithStack(err)
	}

	wfrun, err := w.client.WorkflowRunGet(jobInfo.ProjectKey, jobInfo.WorkflowName, jobInfo.Number)
	if err != nil {
		return err
	}

	for _, it := range wfrun.Workflow.Integrations {
		integrationName := it.ProjectIntegration.Name
		hook, err := w.client.ProjectIntegrationWorkerHookGet(jobInfo.ProjectKey, integrationName)
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		if hook.Disable {
			continue
		}
		if w.cfg.Region != "" {
			if sdk.IsInArray(w.cfg.Region, hook.Configuration.DisableOnRegions) {
				continue
			}
		}

		for capa, hookConfig := range hook.Configuration.ByCapabilities {
			// check is the capabilities exist on the current worker
			if _, err := exec.LookPath(capa); err != nil {
				// The error contains 'Executable file not found', the capa is not on the worker
				continue
			}

			hookFilename := fmt.Sprintf("%d-%s-%s", hookConfig.Priority, integrationName, slug.Convert(hookConfig.Label))

			w.hooks = append(w.hooks, workerHook{
				Config:       hookConfig,
				SetupPath:    path.Join(workingDir, "setup", hookFilename),
				TeardownPath: path.Join(workingDir, "teardown", hookFilename),
			})
		}

		for _, h := range w.hooks {
			infos := []sdk.SpawnInfo{{
				RemoteTime: time.Now(),
				Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerHookSetup.ID, Args: []interface{}{h.Config.Label}},
			}}
			if err := w.Client().QueueJobSendSpawnInfo(ctx, strconv.FormatInt(w.currentJob.wJob.ID, 10), infos); err != nil {
				return sdk.WrapError(err, "cannot record QueueJobSendSpawnInfo for job (err spawn): %d", w.currentJob.wJob.ID)
			}

			log.Info(ctx, "setting up hook at %q", h.SetupPath)

			hookFile, err := fs.Create(h.SetupPath)
			if err != nil {
				return errors.Errorf("unable to open hook file %q in %q: %v", h.SetupPath, w.basedir.Name(), err)
			}
			if _, err := hookFile.WriteString(h.Config.Setup); err != nil {
				_ = hookFile.Close
				return errors.Errorf("unable to setup hook %q: %v", h.SetupPath, err)
			}
			if err := hookFile.Close(); err != nil {
				return errors.Errorf("unable to setup hook %q: %v", h.SetupPath, err)
			}

			hookFile, err = fs.Create(h.TeardownPath)
			if err != nil {
				return errors.Errorf("unable to open hook file %q: %v", h.TeardownPath, err)
			}
			if _, err := hookFile.WriteString(h.Config.Teardown); err != nil {
				_ = hookFile.Close
				return errors.Errorf("unable to setup hook %q: %v", h.TeardownPath, err)
			}
			if err := hookFile.Close(); err != nil {
				return errors.Errorf("unable to setup hook %q: %v", h.TeardownPath, err)
			}
		}
	}
	return nil
}

func (w *CurrentWorker) executeHooksSetup(ctx context.Context, fs afero.Fs, workingDir string) error {
	if strings.EqualFold(runtime.GOOS, "windows") {
		log.Warn(ctx, "hooks are not supported on windows")
		return nil
	}

	var result = make(map[string]string)

	basedir, ok := fs.(*afero.BasePathFs)
	if !ok {
		return sdk.WithStack(fmt.Errorf("invalid given basedir"))
	}

	workerEnv := w.Environ()

	for _, h := range w.hooks {
		filepath, err := basedir.RealPath(h.SetupPath)
		if err != nil {
			return sdk.WrapError(err, "cannot get real path for: %s", h.SetupPath)
		}

		infos := []sdk.SpawnInfo{{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerHookRun.ID, Args: []interface{}{h.Config.Label}},
		}}
		if err := w.Client().QueueJobSendSpawnInfo(ctx, strconv.FormatInt(w.currentJob.wJob.ID, 10), infos); err != nil {
			return sdk.WrapError(err, "cannot record QueueJobSendSpawnInfo for job (err spawn): %d", w.currentJob.wJob.ID)
		}

		str := fmt.Sprintf("source %s ; echo '<<<ENVIRONMENT>>>' ; env", filepath)
		cmd := exec.Command("bash", "-c", str)
		cmd.Env = workerEnv
		bs, err := cmd.CombinedOutput()
		if err != nil {
			return errors.WithStack(err)
		}
		s := bufio.NewScanner(bytes.NewReader(bs))
		start := false
		for s.Scan() {
			if s.Text() == "<<<ENVIRONMENT>>>" {
				start = true
			} else if start {
				kv := strings.SplitN(s.Text(), "=", 2)
				if len(kv) == 2 {
					k := kv[0]
					v := kv[1]
					if !sdk.IsInArray(k+"="+v, workerEnv) {
						result[k] = v
					}
				}
			}
		}
	}
	w.currentJob.envFromHooks = result
	return nil
}

func (w *CurrentWorker) executeHooksTeardown(ctx context.Context, fs afero.Fs, workingDir string) error {
	basedir, ok := fs.(*afero.BasePathFs)
	if !ok {
		return sdk.WithStack(fmt.Errorf("invalid given basedir"))
	}

	for _, h := range w.hooks {
		filepath, err := basedir.RealPath(h.SetupPath)
		if err != nil {
			return sdk.WrapError(err, "cannot get real path for: %s", h.SetupPath)
		}

		infos := []sdk.SpawnInfo{{
			RemoteTime: time.Now(),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerHookRunTeardown.ID, Args: []interface{}{h.Config.Label}},
		}}
		if err := w.Client().QueueJobSendSpawnInfo(ctx, strconv.FormatInt(w.currentJob.wJob.ID, 10), infos); err != nil {
			return sdk.WrapError(err, "cannot record QueueJobSendSpawnInfo for job (err spawn): %d", w.currentJob.wJob.ID)
		}

		cmd := exec.Command("bash", "-c", filepath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return errors.WithMessage(err, w.blur.String(string(output)))
		}
	}
	return nil
}
