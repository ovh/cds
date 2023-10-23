package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
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
	actionContext.Matrix = w.currentJobV2.runJob.Matrix

	// Init step context
	w.currentJobV2.runJob.StepsStatus = sdk.JobStepsStatus{}

	for jobStepIndex, step := range w.currentJobV2.runJob.Job.Steps {
		// Reset step log line to 0
		w.stepLogLine = 0
		w.currentJobV2.currentStepIndex = jobStepIndex
		ctx = workerruntime.SetStepOrder(ctx, jobStepIndex)

		w.currentJobV2.currentStepName = sdk.GetJobStepName(step.ID, jobStepIndex)
		ctx = workerruntime.SetStepName(ctx, w.currentJobV2.currentStepName)

		currentStepStatus := sdk.JobStepStatus{
			Started: time.Now(),
		}
		w.currentJobV2.runJob.StepsStatus[w.currentJobV2.currentStepName] = currentStepStatus
		actionContext.Steps = w.currentJobV2.runJob.StepsStatus.ToStepContext()

		if err := w.ClientV2().V2QueueJobStepUpdate(ctx, w.currentJobV2.runJob.Region, w.currentJobV2.runJob.ID, w.currentJobV2.runJob.StepsStatus); err != nil {
			return w.failJob(ctx, fmt.Sprintf("unable to update step context: %v", err))
		}

		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step %q", w.currentJobV2.currentStepName))
		stepRes := w.runActionStep(ctx, step, w.currentJobV2.currentStepName, actionContext)
		w.SendTerminatedStepLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step %q", w.currentJobV2.currentStepName))
		w.gelfLogger.hook.Flush()

		currentStepStatus.Ended = time.Now()
		currentStepStatus.Outcome = stepRes.Status

		if stepRes.Status == sdk.StatusFail && jobResult.Status != sdk.StatusFail && !step.ContinueOnError {
			jobResult.Status = sdk.StatusFail
			jobResult.Error = stepRes.Error
		}

		if step.ContinueOnError {
			currentStepStatus.Conclusion = sdk.StatusSuccess
		} else {
			currentStepStatus.Conclusion = currentStepStatus.Outcome
		}

		w.currentJobV2.runJob.StepsStatus[w.currentJobV2.currentStepName] = currentStepStatus

		if err := w.ClientV2().V2QueueJobStepUpdate(ctx, w.currentJobV2.runJob.Region, w.currentJobV2.runJob.ID, w.currentJobV2.runJob.StepsStatus); err != nil {
			return w.failJob(ctx, fmt.Sprintf("unable to update step context: %v", err))
		}
	}
	return jobResult
}

func (w *CurrentWorker) runActionStep(ctx context.Context, step sdk.ActionStep, stepName string, runJobContext sdk.WorkflowRunJobsContext) sdk.V2WorkflowRunJobResult {
	if step.If == "" {
		step.If = "${{ success() }}"
	}

	if !strings.HasPrefix(step.If, "${{") {
		step.If = fmt.Sprintf("${{ %s }}", step.If)
	}
	bts, err := json.Marshal(runJobContext)
	if err != nil {
		return sdk.V2WorkflowRunJobResult{
			Status: sdk.StatusFail,
			Time:   time.Now(),
			Error:  fmt.Sprintf("unable to parse step %s condition expression: %v", stepName, err),
		}
	}
	var mapContexts map[string]interface{}
	if err := json.Unmarshal(bts, &mapContexts); err != nil {
		return sdk.V2WorkflowRunJobResult{
			Status: sdk.StatusFail,
			Time:   time.Now(),
			Error:  fmt.Sprintf("unable to parse step %s condition expression: %v", stepName, err),
		}
	}

	ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)
	interpolatedInput, err := ap.Interpolate(ctx, step.If)
	if err != nil {
		return sdk.V2WorkflowRunJobResult{
			Status: sdk.StatusFail,
			Time:   time.Now(),
			Error:  fmt.Sprintf("unable to interpolate step condition %s: %v", step.If, err),
		}
	}

	if _, ok := interpolatedInput.(string); !ok {
		return sdk.V2WorkflowRunJobResult{
			Status: sdk.StatusFail,
			Time:   time.Now(),
			Error:  fmt.Sprintf("step %s: if statement does not return a string. Got %v", stepName, interpolatedInput),
		}
	}

	booleanResult, err := strconv.ParseBool(interpolatedInput.(string))
	if err != nil {
		return sdk.V2WorkflowRunJobResult{
			Status: sdk.StatusFail,
			Time:   time.Now(),
			Error:  fmt.Sprintf("step %s: if statement does not return a boolean. Got %v", stepName, interpolatedInput),
		}
	}

	if !booleanResult {
		return sdk.V2WorkflowRunJobResult{
			Status: sdk.StatusSkipped,
			Time:   time.Now(),
		}
	}

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
	stepName := parentStepName + "-" + filepath.Base(name)

	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step \"%s\"", stepName))
	defer func() {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step \"%s\"", stepName))
	}()

	// Set action inputs
	inputs := make(map[string]string)
	if len(actionPath) == 1 {
		p := w.GetActionPlugin(actionPath[0])
		if p == nil {
			var err error
			p, err = w.PluginGet(actionPath[0])
			if err != nil {
				return w.failJob(ctx, fmt.Sprintf("unable to retrieve plugin %s: %v", actionPath[0], err))
			}
			w.SetActionPlugin(p)
		}
		for k, inp := range p.Inputs {
			inputs[k] = inp.Default
		}
	} else {
		actionDef, found := w.actions[name]
		if !found {
			return w.failJob(ctx, fmt.Sprintf("action %s not found", name))
		}
		for k, inp := range actionDef.Inputs {
			inputs[k] = inp.Default
		}
	}
	for k, with := range inputWith {
		if _, has := inputs[k]; has {
			inputs[k] = with
		}
	}
	// Compute context
	actionContext, err := w.computeContextForAction(ctx, parentContext, inputs)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("unable to compute context for action %s: %v", name, err))
	}

	switch len(actionPath) {
	case 1:
		opts := make(map[string]string, 0)
		for k, v := range actionContext.Inputs {
			if vString, ok := v.(string); ok {
				opts[k] = vString
			}
		}
		return w.runPlugin(ctx, actionPath[0], opts)
	case 5:
		for stepIndex, step := range w.actions[name].Runs.Steps {
			stepRes := w.runSubActionStep(ctx, step, stepName, stepIndex, actionContext)
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

func (w *CurrentWorker) runSubActionStep(ctx context.Context, step sdk.ActionStep, stepName string, stepIndex int, runJobContext sdk.WorkflowRunJobsContext) sdk.V2WorkflowRunJobResult {
	currentStep := stepName + "-" + sdk.GetJobStepName(step.ID, stepIndex)
	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step %q", currentStep))
	defer func() {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step %q", currentStep))
	}()
	return w.runActionStep(ctx, step, currentStep, runJobContext)
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
	pluginClient, err := w.pluginFactory.NewClient(ctx, w, plugin.TypeAction, pluginName, plugin.InputManagementStrict)
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
