package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/internal/plugin"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func (w *CurrentWorker) V2ProcessJob() (res sdk.V2WorkflowRunJobResult) {
	ctx := w.currentJobV2.context
	t0 := time.Now()

	// Timeout must be the same as the goroutine which stop jobs in package api/workflow
	ctx, cancel := context.WithTimeout(ctx, 24*time.Hour)
	log.Info(ctx, "Process Job %s (%s)", w.currentJobV2.runJob.JobID, w.currentJobV2.runJob.ID)
	defer func() {
		log.Info(ctx, "Process Job Done %s (%s) :%s", w.currentJobV2.runJob.JobID, w.currentJobV2.runJob.ID, sdk.Round(time.Since(t0), time.Second).String())
	}()
	defer cancel()

	ctx = workerruntime.SetRunJobID(ctx, w.currentJobV2.runJob.ID)
	ctx = workerruntime.SetStepOrder(ctx, 0)
	defer func() {
		if res.Status == sdk.StatusSuccess {
			log.Warn(ctx, "Status: %s", res.Status)
		} else {
			log.Warn(ctx, "Status: %s | Reason: %s", res.Status, res.Error)
		}
	}()

	wdFile, wdAbs, err := w.setupWorkingDirectory(ctx, w.currentJobV2.runJob.JobID)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return w.failJob(ctx, fmt.Sprintf("Error: unable to setup workfing directory: %v", err))
	}
	w.workingDirAbs = wdAbs
	ctx = workerruntime.SetWorkingDirectory(ctx, wdFile)
	log.Debug(ctx, "Setup workspace - %s", wdFile.Name())

	kdFile, _, err := w.setupKeysDirectory(ctx, w.currentJobV2.runJob.JobID)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return w.failJob(ctx, fmt.Sprintf("Error: unable to setup keys directory: %v", err))
	}
	ctx = workerruntime.SetKeysDirectory(ctx, kdFile)
	log.Debug(ctx, "Setup key directory - %s", kdFile.Name())

	tdFile, _, err := w.setupTmpDirectory(ctx, w.currentJobV2.runJob.JobID)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("Error: unable to setup tmp directory: %v", err))
	}
	ctx = workerruntime.SetTmpDirectory(ctx, tdFile)
	log.Debug(ctx, "Setup tmp directory - %s", tdFile.Name())

	hdFile, _, err := w.setupHooksDirectory(ctx, w.currentJobV2.runJob.JobID)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return w.failJob(ctx, fmt.Sprintf("Error: unable to setup hooks directory: %v", err))
	}

	// TODO w.setupHooks

	w.currentJobV2.context = ctx
	w.currentJobV2.runJobContext.CDS.Workspace = wdAbs

	log.Info(ctx, "Executing hooks setup from directory: %s", hdFile.Name())
	if err := w.executeHooksSetup(ctx, w.basedir, hdFile.Name()); err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return w.failJob(ctx, fmt.Sprintf("Error: unable to setup hooks: %v", err))
	}
	res = w.runJobAsCode(ctx)

	// TODO Teardown worker hooks

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

func (w *CurrentWorker) runJobAsCode(ctx context.Context) sdk.V2WorkflowRunJobResult {
	log.Info(ctx, "runJob> start job %s (%d)", w.currentJobV2.runJob.JobID, w.currentJobV2.runJob.ID)
	var jobResult = sdk.V2WorkflowRunJobResult{
		Status: sdk.StatusSuccess,
	}

	defer func() {
		w.gelfLogger.hook.Flush()
		log.Info(ctx, "runJob> end of job %s (%s)", w.currentJobV2.runJob.JobID, w.currentJobV2.runJob.ID)
	}()

	actionContext, err := w.computeContextForAction(ctx, w.currentJobV2.runJobContext, w.currentJobV2.runJob.Job.Inputs)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("unable to compute job context: %v", err))
	}

	// Init step context
	stepsContext := sdk.StepsContext{}

	for jobStepIndex, step := range w.currentJobV2.runJob.Job.Steps {
		// Reset step log line to 0
		w.stepLogLine = 0
		w.currentJobV2.currentStepIndex = jobStepIndex
		ctx = workerruntime.SetStepOrder(ctx, jobStepIndex)

		w.currentJobV2.currentStepName = sdk.GetJobStepName(step.ID, jobStepIndex)
		ctx = workerruntime.SetStepName(ctx, w.currentJobV2.currentStepName)

		stepsContext[w.currentJobV2.currentStepName] = sdk.StepContext{}
		actionContext.Steps = stepsContext

		stepRes := w.runActionStep(ctx, step, w.currentJobV2.currentStepName, actionContext)

		stepCxt := actionContext.Steps[w.currentJobV2.currentStepName]
		stepCxt.Outcome = stepRes.Status
		// FIXME - continue-on-error
		stepCxt.Conclusion = stepRes.Status
		actionContext.Steps[w.currentJobV2.currentStepName] = stepCxt

		if err := w.ClientV2().V2QueueJobStepUpdate(ctx, w.currentJobV2.runJob.Region, w.currentJobV2.runJob.ID, actionContext.Steps); err != nil {
			return w.failJob(ctx, fmt.Sprintf("unable to update step context: %v", err))
		}

		if stepRes.Status == sdk.StatusFail {
			return stepRes
		}
	}
	return jobResult
}

func (w *CurrentWorker) runActionStep(ctx context.Context, step sdk.ActionStep, stepName string, runJobContext sdk.WorkflowRunJobsContext) sdk.V2WorkflowRunJobResult {
	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step %q", stepName))
	defer func() {
		w.SendTerminatedStepLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step %q", stepName))
		w.gelfLogger.hook.Flush()
	}()

	// TODO manage step if condition

	var result sdk.V2WorkflowRunJobResult
	switch {
	case step.Uses != "":
		result = w.runJobStepAction(ctx, step, runJobContext, stepName, step.With)
	case step.Run != "":
		result = w.runJobStepScript(ctx, step, runJobContext)
	default:
		return w.failJob(ctx, "invalid action definition. Missing uses or run keys")
	}
	return result
}

func (w *CurrentWorker) runJobStepAction(ctx context.Context, step sdk.ActionStep, parentContext sdk.WorkflowRunJobsContext, parentStepName string, inputWith map[string]string) sdk.V2WorkflowRunJobResult {
	name := strings.TrimPrefix(step.Uses, "actions/")
	actionPath := strings.Split(name, "/")
	stepName := parentStepName + filepath.Base(name)

	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step \"%s\"", stepName))
	defer func() {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step \"%s\"", stepName))
	}()

	actionDef, found := w.actions[name]
	if !found {
		return w.failJob(ctx, fmt.Sprintf("action %s not found", name))
	}

	// Compute context for action
	inputs := make(map[string]string)
	for k, inp := range actionDef.Inputs {
		inputs[k] = inp.Default
	}
	for k, with := range inputWith {
		if _, has := inputs[k]; has {
			inputs[k] = with
		}
	}
	actionContext, err := w.computeContextForAction(ctx, parentContext, inputs)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("unable to compute context for action %s: %v", name, err))
	}

	switch len(actionPath) {
	case 1:
		// plugin
		opts := make(map[string]string, 0)
		for k, v := range actionContext.Inputs {
			vString, ok := v.(string)
			if !ok {
				return w.failJob(ctx, fmt.Sprintf("input %s is not a string. Got %T", k, v))
			}
			opts[k] = vString
		}
		return w.runPlugin(ctx, actionPath[0], opts)
	case 5:
		for _, step := range actionDef.Runs.Steps {
			stepRes := w.runActionStep(ctx, step, w.currentJobV2.currentStepName, actionContext)
			if stepRes.Status == sdk.StatusFail {
				return stepRes
			}
		}
	default:
	}
	return sdk.V2WorkflowRunJobResult{
		Status: sdk.StatusSuccess,
	}
}

func (w *CurrentWorker) runJobStepScript(ctx context.Context, step sdk.ActionStep, runJobContext sdk.WorkflowRunJobsContext) sdk.V2WorkflowRunJobResult {
	bts, err := json.Marshal(runJobContext)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("unable to marshal contexts: %v", err))
	}
	var mapContexts map[string]interface{}
	if err := json.Unmarshal(bts, &mapContexts); err != nil {
		return w.failJob(ctx, fmt.Sprintf("unable to unmarshal contexts: %v", err))
	}

	interpolatedInput, err := w.interpolateActionInput(ctx, mapContexts, step.Run)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("unable to interpolate script content: %v", err))
	}
	contentString, ok := interpolatedInput.(string)
	if !ok {
		return w.failJob(ctx, fmt.Sprintf("interpolated script content is not a string. Got %T", interpolatedInput))
	}
	return w.runPlugin(ctx, "script", map[string]string{
		"content": contentString,
	})
}

func (w *CurrentWorker) runPlugin(ctx context.Context, pluginName string, opts map[string]string) sdk.V2WorkflowRunJobResult {
	pluginClient, err := plugin.NewClient(ctx, w, plugin.TypeAction, pluginName)
	if pluginClient != nil {
		defer pluginClient.Close(ctx)
	}
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("%v", err))
	}

	pluginResult := pluginClient.Run(ctx, opts)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("error runnning artifact %s: %v", pluginName, err))
	}

	if pluginResult.Status == sdk.StatusFail {
		return w.failJob(ctx, pluginResult.Details)
	}

	return sdk.V2WorkflowRunJobResult{
		Status: pluginResult.Status,
		Error:  pluginResult.Details,
	}
}

func (w *CurrentWorker) failJob(ctx context.Context, reason string) sdk.V2WorkflowRunJobResult {
	res := sdk.V2WorkflowRunJobResult{
		Status: sdk.StatusFail,
		Error:  reason,
	}
	log.Error(ctx, "worker.failJobStep> %v", res.Error)
	w.SendLog(ctx, workerruntime.LevelError, res.Error)
	return res
}

func (w *CurrentWorker) interpolateActionInput(ctx context.Context, contexts map[string]interface{}, input string) (interface{}, error) {
	ap := sdk.NewActionParser(contexts, sdk.DefaultFuncs)
	interpolatedInput, err := ap.Interpolate(ctx, input)
	if err != nil {
		return "", err
	}
	return interpolatedInput, nil
}

func (w *CurrentWorker) computeContextForAction(ctx context.Context, parentContext sdk.WorkflowRunJobsContext, inputs map[string]string) (sdk.WorkflowRunJobsContext, error) {
	if len(inputs) == 0 {
		return parentContext, nil
	}
	actionContext := sdk.WorkflowRunJobsContext{
		Inputs: make(map[string]interface{}),
		Steps:  parentContext.Steps,
	}

	mapContextBytes, _ := json.Marshal(parentContext)
	var mapContext map[string]interface{}
	if err := json.Unmarshal(mapContextBytes, &mapContext); err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return actionContext, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to read job context")
	}

	for k, v := range inputs {
		interpolatedInput, err := w.interpolateActionInput(ctx, mapContext, v)
		if err != nil {
			return actionContext, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to interpolate job inputs: %v", err)
		}
		actionContext.Inputs[k] = interpolatedInput
	}
	w.currentJobV2.runJobContext = actionContext
	return actionContext, nil
}
