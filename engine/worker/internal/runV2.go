package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rockbears/log"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/internal/plugin"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/slug"
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
		if res.Status == sdk.V2WorkflowRunJobStatusSuccess {
			log.Warn(ctx, "Status: %s", res.Status)
		} else {
			log.Warn(ctx, "Status: %s | Reason: %s", res.Status, res.Error)
		}
	}()

	wdFile, wdAbs, err := w.setupWorkingDirectory(ctx, w.currentJobV2.runJob.JobID)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return w.failJob(ctx, fmt.Sprintf("Error: unable to setup working directory: %v", err))
	}
	w.workingDirAbs = wdAbs
	ctx = workerruntime.SetWorkingDirectory(ctx, wdFile)
	log.Debug(ctx, "Setup workspace - %s", wdFile.Name())

	// Manage services readiness
	if result := w.runJobServicesReadiness(ctx); result.Status != sdk.V2WorkflowRunJobStatusSuccess {
		return w.failJob(ctx, fmt.Sprintf("Error: readiness service command failed: %v", result.Error))
	}

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

	if err := w.setupHooksV2(ctx, w.currentJobV2, w.basedir, hdFile.Name()); err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return w.failJob(ctx, fmt.Sprintf("Error: unable to setup hooks: %v", err))
	}

	w.currentJobV2.context = ctx
	w.currentJobV2.runJobContext.CDS.Workspace = wdAbs

	log.Info(ctx, "Executing hooks setup from directory: %s", hdFile.Name())
	if err := w.executeHooksSetupV2(ctx, w.basedir); err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return w.failJob(ctx, fmt.Sprintf("Error: unable to setup hooks: %v", err))
	}
	res = w.runJobAsCode(ctx)

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

func (w *CurrentWorker) runJobServicesReadiness(ctx context.Context) sdk.V2WorkflowRunJobResult {
	ctx = workerruntime.SetIsReadinessServices(ctx, true)
	defer workerruntime.SetIsReadinessServices(ctx, false)

	result := sdk.V2WorkflowRunJobResult{Status: sdk.V2WorkflowRunJobStatusSuccess}
	if w.currentJobV2.runJob.Job.Services == nil {
		return result
	}

	for serviceName, service := range w.currentJobV2.runJob.Job.Services {
		if service.Readiness.Command == "" {
			continue
		}

		if err := w.runJobServiceReadiness(ctx, serviceName, service); err != nil {
			result.Error = fmt.Sprintf("failed on check service readiness: %v", err.Error())
			result.Status = sdk.V2WorkflowRunJobStatusFail
			return result
		}
	}
	return result
}

func (w *CurrentWorker) computeContextForJob(ctx context.Context) error {
	// Interpolate env ctx and input ctx

	mapContextBts, _ := json.Marshal(w.currentJobV2.runJobContext)
	var parserContext map[string]interface{}
	if err := json.Unmarshal(mapContextBts, &parserContext); err != nil {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid context found: %v", err)
	}
	ap := sdk.NewActionParser(parserContext, sdk.DefaultFuncs)
	for k, e := range w.currentJobV2.runJobContext.Env {
		interpolatedValue, err := ap.InterpolateToString(ctx, e)
		if err != nil {
			return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to interpolate env variabke %s: %v", k, err)
		}
		w.currentJobV2.runJobContext.Env[k] = interpolatedValue
	}

	// Interpolate inputs if needed
	if len(w.currentJobV2.runJobContext.Inputs) > 0 {
		mapContextBts, _ := json.Marshal(w.currentJobV2.runJobContext)
		var parserContext map[string]interface{}
		if err := json.Unmarshal(mapContextBts, &parserContext); err != nil {
			return sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid context found: %v", err)
		}
		ap := sdk.NewActionParser(parserContext, sdk.DefaultFuncs)
		for k, e := range w.currentJobV2.runJobContext.Inputs {
			interpolatedValue, err := ap.InterpolateToString(ctx, e)
			if err != nil {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to interpolate env variabke %s: %v", k, err)
			}
			w.currentJobV2.runJobContext.Inputs[k] = interpolatedValue

		}
	}
	return nil
}

func (w *CurrentWorker) runJobServiceReadiness(ctx context.Context, serviceName string, service sdk.V2JobService) error {
	step := sdk.ActionStep{
		Run: service.Readiness.Command,
	}
	runJobContext := sdk.WorkflowRunJobsContext{
		WorkflowRunContext: sdk.WorkflowRunContext{
			Env: service.Env,
		},
	}

	interval, err := time.ParseDuration(service.Readiness.Interval)
	if err != nil {
		return fmt.Errorf("unable to parse interval %q", interval)
	}
	timeout, err := time.ParseDuration(service.Readiness.Timeout)
	if err != nil {
		return fmt.Errorf("unable to parse timeout %q", timeout)
	}

	if service.Readiness.Retries <= 0 {
		return fmt.Errorf("retries value must be > 0 (current: %d)", service.Readiness.Retries)
	}

	for i := 0; i < service.Readiness.Retries; i++ {
		ctxA, cancel := context.WithTimeout(ctx, timeout)
		result := w.runJobStepScript(ctxA, step, runJobContext)
		cancel()

		info := sdk.V2SendJobRunInfo{
			Time: time.Now(),
		}

		if result.Status == sdk.V2WorkflowRunJobStatusSuccess {
			info.Level = sdk.WorkflowRunInfoLevelInfo
			info.Message = fmt.Sprintf("service %s is ready", serviceName)
		} else {
			info.Level = sdk.WorkflowRunInfoLevelWarning
			info.Message = fmt.Sprintf("service %s is not ready (%s)", serviceName, result.Status)
		}

		if err := w.ClientV2().V2QueuePushJobInfo(ctx, w.currentJobV2.runJob.Region, w.currentJobV2.runJob.ID, info); err != nil {
			log.Error(ctx, "runJobServiceReadiness> Unable to send spawn info: %v", err)
		}

		if result.Status == sdk.V2WorkflowRunJobStatusSuccess {
			return nil
		}

		time.Sleep(interval)
	}

	info := sdk.V2SendJobRunInfo{
		Message: fmt.Sprintf("service %s fails to be ready", serviceName),
		Level:   sdk.WorkflowRunInfoLevelError,
		Time:    time.Now(),
	}

	if err := w.ClientV2().V2QueuePushJobInfo(ctx, w.currentJobV2.runJob.Region, w.currentJobV2.runJob.ID, info); err != nil {
		log.Error(ctx, "runJobServiceReadiness> Unable to send spawn info: %v", err)
	}

	return fmt.Errorf("readiness service %s: Failed after %d retries", serviceName, service.Readiness.Retries)
}

func (w *CurrentWorker) runJobAsCode(ctx context.Context) sdk.V2WorkflowRunJobResult {
	log.Info(ctx, "runJob> start job %s (%s)", w.currentJobV2.runJob.JobID, w.currentJobV2.runJob.ID)
	var jobResult = sdk.V2WorkflowRunJobResult{
		Status: sdk.V2WorkflowRunJobStatusSuccess,
	}

	defer func() {
		w.gelfLogger.hook.Flush()
		log.Info(ctx, "runJob> end of job %s (%s)", w.currentJobV2.runJob.JobID, w.currentJobV2.runJob.ID)
	}()

	// Interpolate job context
	if err := w.computeContextForJob(ctx); err != nil {
		return w.failJob(ctx, fmt.Sprintf("unable to compute job context: %v", err))
	}

	// Init step context
	w.currentJobV2.runJob.StepsStatus = sdk.JobStepsStatus{}

	for jobStepIndex, step := range w.currentJobV2.runJob.Job.Steps {
		// Reset step log line to 0
		w.stepLogLine = 0
		w.currentJobV2.currentStepIndex = jobStepIndex
		ctx = workerruntime.SetStepOrder(ctx, jobStepIndex)

		// Set step in context
		w.currentJobV2.currentStepName = sdk.GetJobStepName(step.ID, jobStepIndex)
		ctx = workerruntime.SetStepName(ctx, w.currentJobV2.currentStepName)

		// Set current step status + create step context
		w.createStepStatus(w.currentJobV2.runJob.StepsStatus, w.currentJobV2.currentStepName)
		w.currentJobV2.runJobContext.Steps = w.currentJobV2.runJob.StepsStatus.ToStepContext()

		if err := w.ClientV2().V2QueueJobStepUpdate(ctx, w.currentJobV2.runJob.Region, w.currentJobV2.runJob.ID, w.currentJobV2.runJob.StepsStatus); err != nil {
			return w.failJob(ctx, fmt.Sprintf("unable to update step context: %v", err))
		}
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step %q", w.currentJobV2.currentStepName))

		// Step context = parent context + env set on step
		currentStepContext, err := w.createStepContext(ctx, step, w.currentJobV2.runJobContext)
		if err != nil {
			w.failJob(ctx, err.Error())
		}
		stepRes := w.runActionStep(ctx, step, w.currentJobV2.currentStepName, *currentStepContext)
		w.SendTerminatedStepLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step %q", w.currentJobV2.currentStepName))
		w.gelfLogger.hook.Flush()

		w.updateStepResult(w.currentJobV2.runJob.StepsStatus, &jobResult, stepRes, step, w.currentJobV2.currentStepName)

		if err := w.ClientV2().V2QueueJobStepUpdate(ctx, w.currentJobV2.runJob.Region, w.currentJobV2.runJob.ID, w.currentJobV2.runJob.StepsStatus); err != nil {
			return w.failJob(ctx, fmt.Sprintf("unable to update step context: %v", err))
		}
	}
	return jobResult
}

func (w *CurrentWorker) createStepStatus(stepsStatus sdk.JobStepsStatus, stepName string) {
	currentStepStatus := sdk.JobStepStatus{
		Started: time.Now(),
	}
	stepsStatus[stepName] = currentStepStatus
}

func (w *CurrentWorker) createStepContext(ctx context.Context, step sdk.ActionStep, parentContext sdk.WorkflowRunJobsContext) (*sdk.WorkflowRunJobsContext, error) {
	// Step context = parent context + step.env
	currentStepContext := sdk.WorkflowRunJobsContext{
		WorkflowRunContext: sdk.WorkflowRunContext{
			CDS: parentContext.CDS,
			Git: parentContext.Git,
			Env: make(map[string]string),
		},
		Inputs:       parentContext.Inputs,
		Jobs:         parentContext.Jobs,
		Needs:        parentContext.Needs,
		Steps:        parentContext.Steps,
		Matrix:       parentContext.Matrix,
		Integrations: parentContext.Integrations,
		Gate:         parentContext.Gate,
		Vars:         parentContext.Vars,
	}

	// Copy parent env context
	currentStepContext.Env = make(map[string]string)
	for k, v := range parentContext.Env {
		currentStepContext.Env[k] = v
	}

	// Add step.env in step context
	if len(step.Env) > 0 {
		mapContextBts, _ := json.Marshal(parentContext)
		var parserContext map[string]interface{}
		if err := json.Unmarshal(mapContextBts, &parserContext); err != nil {
			return nil, fmt.Errorf("unable to read context: %v", err)
		}
		ap := sdk.NewActionParser(parserContext, sdk.DefaultFuncs)
		for k, v := range step.Env {
			// Interpolate current step env variable
			resultString, err := ap.InterpolateToString(ctx, v)
			if err != nil {
				return nil, fmt.Errorf("unable to interpolate env variable %s [%s]: %v", k, v, err)
			}
			currentStepContext.Env[k] = resultString
		}
	}
	return &currentStepContext, nil
}

func (w *CurrentWorker) canRunStep(ctx context.Context, stepName string, stepCondition string, stepContext sdk.WorkflowRunJobsContext) (bool, error) {
	bts, err := json.Marshal(stepContext)
	if err != nil {
		return false, fmt.Errorf("unable to parse step %s condition expression: %v", stepName, err)
	}
	var mapContexts map[string]interface{}
	if err := json.Unmarshal(bts, &mapContexts); err != nil {
		return false, fmt.Errorf("unable to parse step %s condition expression: %v", stepName, err)
	}

	if stepCondition == "" {
		stepCondition = "${{ success() }}"
	}

	if !strings.HasPrefix(stepCondition, "${{") {
		stepCondition = fmt.Sprintf("${{ %s }}", stepCondition)
	}

	ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)
	booleanResult, err := ap.InterpolateToBool(ctx, stepCondition)
	if err != nil {
		return false, fmt.Errorf("unable to interpolate step condition %s on step %s into a boolean: %v", stepCondition, stepName, err)
	}
	return booleanResult, nil
}

func (w *CurrentWorker) runActionStep(ctx context.Context, step sdk.ActionStep, stepName string, currentContext sdk.WorkflowRunJobsContext) sdk.V2WorkflowRunJobResult {
	booleanResult, err := w.canRunStep(ctx, stepName, step.If, currentContext)
	if err != nil {
		return sdk.V2WorkflowRunJobResult{
			Status: sdk.V2WorkflowRunJobStatusFail,
			Time:   time.Now(),
			Error:  fmt.Sprintf("%v", err),
		}
	}

	if !booleanResult {
		w.SendLog(ctx, workerruntime.LevelInfo, "not executed")
		return sdk.V2WorkflowRunJobResult{
			Status: sdk.V2WorkflowRunJobStatusSkipped,
			Time:   time.Now(),
		}
	}

	var result sdk.V2WorkflowRunJobResult
	switch {
	case step.Uses != "":
		result = w.runJobStepAction(ctx, step, currentContext, stepName, step.With)
	case step.Run != "":
		result = w.runJobStepScript(ctx, step, currentContext)
	default:
		return w.failJob(ctx, "invalid action definition. Missing uses or run keys")
	}
	return result
}

func (w *CurrentWorker) runJobStepAction(ctx context.Context, step sdk.ActionStep, currentStepContext sdk.WorkflowRunJobsContext, parentStepName string, inputWith map[string]string) sdk.V2WorkflowRunJobResult {
	name := strings.TrimPrefix(step.Uses, "actions/")
	actionRefSplit := strings.Split(name, "@")
	actionPath := strings.Split(actionRefSplit[0], "/")
	stepName := parentStepName + "-" + filepath.Base(name)

	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step \"%s\"", stepName))
	defer func() {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step \"%s\"", stepName))
	}()

	actionResult := sdk.V2WorkflowRunJobResult{
		Status: sdk.V2WorkflowRunJobStatusSuccess,
	}

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

	// Compute action/plugin context
	actionContext, err := w.computeContextForAction(ctx, currentStepContext, inputs, step)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("unable to compute context for action %s: %v", name, err))
	}

	switch len(actionPath) {
	case 1:
		env, err := w.GetEnvVariable(ctx, *actionContext)
		if err != nil {
			return w.failJob(ctx, fmt.Sprintf("%v", err))
		}
		return w.runPlugin(ctx, actionPath[0], actionContext.Inputs, env)
	case 5:
		// <project_key> / vcs / my / repo / actionName
		subStepStatus := sdk.JobStepsStatus{}
		for stepIndex, step := range w.actions[name].Runs.Steps {

			// Create dedicated step status for the given action and set it on his context
			subStepName := sdk.GetJobStepName(step.ID, stepIndex)
			w.createStepStatus(subStepStatus, subStepName)
			actionContext.Steps = subStepStatus.ToStepContext()

			// Create step context: (parent context + current set env)
			currentStepContext, err := w.createStepContext(ctx, step, *actionContext)
			if err != nil {
				w.failJob(ctx, err.Error())
			}
			stepRes := w.runSubActionStep(ctx, step, stepName, stepIndex, *currentStepContext)
			if stepRes.Status == sdk.V2WorkflowRunJobStatusFail {
				return stepRes
			}
			w.updateStepResult(subStepStatus, &actionResult, stepRes, step, subStepName)
		}
	default:
	}
	return sdk.V2WorkflowRunJobResult{
		Status: sdk.V2WorkflowRunJobStatusSuccess,
	}
}

func (w *CurrentWorker) updateStepResult(stepStatus sdk.JobStepsStatus, actionResult *sdk.V2WorkflowRunJobResult, stepRes sdk.V2WorkflowRunJobResult, step sdk.ActionStep, stepName string) {
	currentStepStatus := stepStatus[stepName]
	currentStepStatus.Ended = time.Now()
	currentStepStatus.Outcome = stepRes.Status
	if stepRes.Status == sdk.V2WorkflowRunJobStatusFail && actionResult.Status != sdk.V2WorkflowRunJobStatusFail && !step.ContinueOnError {
		actionResult.Status = sdk.V2WorkflowRunJobStatusFail
		actionResult.Error = stepRes.Error
	}

	if step.ContinueOnError {
		currentStepStatus.Conclusion = sdk.V2WorkflowRunJobStatusSuccess
	} else {
		currentStepStatus.Conclusion = currentStepStatus.Outcome
	}
	stepStatus[w.currentJobV2.currentStepName] = currentStepStatus
}

func (w *CurrentWorker) runSubActionStep(ctx context.Context, step sdk.ActionStep, stepName string, stepIndex int, runJobContext sdk.WorkflowRunJobsContext) sdk.V2WorkflowRunJobResult {
	currentStep := stepName + "-" + sdk.GetJobStepName(step.ID, stepIndex)
	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step %q", currentStep))
	defer func() {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step %q", currentStep))
	}()
	return w.runActionStep(ctx, step, currentStep, runJobContext)
}

func (w *CurrentWorker) runJobStepScript(ctx context.Context, step sdk.ActionStep, stepContext sdk.WorkflowRunJobsContext) sdk.V2WorkflowRunJobResult {
	bts, err := json.Marshal(stepContext)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("unable to marshal contexts: %v", err))
	}
	var mapContexts map[string]interface{}
	if err := json.Unmarshal(bts, &mapContexts); err != nil {
		return w.failJob(ctx, fmt.Sprintf("unable to unmarshal contexts: %v", err))
	}

	ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)
	contentString, err := ap.InterpolateToString(ctx, step.Run)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("unable to interpolate script content: %v", err))
	}

	env, err := w.GetEnvVariable(ctx, stepContext)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("%v", err))
	}
	return w.runPlugin(ctx, "script", map[string]string{
		"content": contentString,
	}, env)
}

func (w *CurrentWorker) runPlugin(ctx context.Context, pluginName string, opts map[string]string, env map[string]string) sdk.V2WorkflowRunJobResult {
	pluginClient, err := w.pluginFactory.NewClient(ctx, w, plugin.TypeStream, pluginName, plugin.InputManagementStrict, env)
	if pluginClient != nil {
		defer pluginClient.Close(ctx)
	}
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("%v", err))
	}

	pluginResult := pluginClient.Run(ctx, opts)

	if pluginResult.Status == sdk.StatusFail {
		return w.failJob(ctx, pluginResult.Details)
	}

	jobStatus, err := sdk.NewV2WorkflowRunJobStatusFromString(pluginResult.Status)
	if err != nil {
		return w.failJob(ctx, fmt.Sprintf("error running plugin %s: %v", pluginName, err))
	}

	return sdk.V2WorkflowRunJobResult{
		Status: jobStatus,
		Error:  pluginResult.Details,
	}
}

func (w *CurrentWorker) failJob(ctx context.Context, reason string) sdk.V2WorkflowRunJobResult {
	res := sdk.V2WorkflowRunJobResult{
		Status: sdk.V2WorkflowRunJobStatusFail,
		Error:  reason,
	}
	log.Error(ctx, "worker.failJobStep> %v", res.Error)
	w.SendLog(ctx, workerruntime.LevelError, res.Error)
	return res
}

// For actions, only pass CDS/GIT/ENV/Integration context from parrent
func (w *CurrentWorker) computeContextForAction(ctx context.Context, parentContext sdk.WorkflowRunJobsContext, inputs map[string]string, step sdk.ActionStep) (*sdk.WorkflowRunJobsContext, error) {
	actionContext := sdk.WorkflowRunJobsContext{
		WorkflowRunContext: sdk.WorkflowRunContext{
			CDS: parentContext.CDS,
			Git: parentContext.Git,
			Env: map[string]string{},
		},
		Integrations: parentContext.Integrations,
		Inputs:       make(map[string]string),
	}

	for k, v := range parentContext.Env {
		actionContext.Env[k] = v
	}
	if len(step.Env) > 0 {
		mapContextBts, _ := json.Marshal(parentContext)
		var parserContext map[string]interface{}
		if err := json.Unmarshal(mapContextBts, &parserContext); err != nil {
			return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid context found: %v", err)
		}
		ap := sdk.NewActionParser(parserContext, sdk.DefaultFuncs)
		for k, e := range step.Env {
			interpolatedValue, err := ap.InterpolateToString(ctx, e)
			if err != nil {
				return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to interpolate env variabke %s: %v", k, err)
			}
			actionContext.Env[k] = interpolatedValue
		}
	}

	if len(inputs) > 0 {
		mapContextBts, _ := json.Marshal(actionContext)
		var parserContext map[string]interface{}
		if err := json.Unmarshal(mapContextBts, &parserContext); err != nil {
			return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "invalid context found: %v", err)
		}
		ap := sdk.NewActionParser(parserContext, sdk.DefaultFuncs)

		// Interpolate step inputs and set them in context
		for k, e := range inputs {
			interpolatedValue, err := ap.InterpolateToString(ctx, e)
			if err != nil {
				return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to interpolate env variabke %s: %v", k, err)
			}
			actionContext.Inputs[k] = interpolatedValue
		}
	}
	return &actionContext, nil
}

func (w *CurrentWorker) GetEnvVariable(ctx context.Context, contexts sdk.WorkflowRunJobsContext) (map[string]string, error) {
	newEnvVar := make(map[string]string)

	var mapCDS map[string]interface{}
	btsCDS, _ := json.Marshal(contexts.CDS)
	if err := json.Unmarshal(btsCDS, &mapCDS); err != nil {
		return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to unmarshal cds context: %v", err)
	}
	for k, v := range mapCDS {
		if strings.EqualFold(k, "event") {
			continue
		}
		switch reflect.ValueOf(v).Kind() {
		case reflect.Map, reflect.Slice:
			s, _ := json.Marshal(v)
			newEnvVar[fmt.Sprintf("CDS_%s", strings.ToUpper(k))] = sdk.OneLineValue(string(s))
		default:
			newEnvVar[fmt.Sprintf("CDS_%s", strings.ToUpper(k))] = sdk.OneLineValue(fmt.Sprintf("%v", v))
		}

	}

	var mapGIT map[string]interface{}
	btsGIT, _ := json.Marshal(contexts.Git)
	if err := json.Unmarshal(btsGIT, &mapGIT); err != nil {
		return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to unmarshal git context")
	}
	for k, v := range mapGIT {
		if strings.EqualFold(k, "changesets") || strings.EqualFold(k, "ssh_private") || strings.EqualFold(k, "token") {
			continue
		}
		newEnvVar[fmt.Sprintf("GIT_%s", strings.ToUpper(k))] = sdk.OneLineValue(fmt.Sprintf("%v", v))
	}

	var mapEnv map[string]interface{}
	btsEnv, _ := json.Marshal(contexts.Env)
	if err := json.Unmarshal(btsEnv, &mapEnv); err != nil {
		return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to unmarshal env context")
	}
	for k, v := range mapEnv {
		if strings.HasPrefix(k, "CDS_") || strings.HasPrefix(k, "GIT_") {
			continue
		}
		newEnvVar[strings.ToUpper(k)] = sdk.OneLineValue(fmt.Sprintf("%v", v))
	}

	// Integration variable
	if w.currentJobV2.runJobContext.Integrations != nil && w.currentJobV2.runJobContext.Integrations.ArtifactManager.Name != "" {
		envVars := computeIntegrationConfigToEnvVar(w.currentJobV2.runJobContext.Integrations.ArtifactManager, "ARTIFACT_MANAGER")
		for k, v := range envVars {
			newEnvVar[k] = v
		}
	}
	if w.currentJobV2.runJobContext.Integrations != nil && w.currentJobV2.runJobContext.Integrations.Deployment.Name != "" {
		envVars := computeIntegrationConfigToEnvVar(w.currentJobV2.runJobContext.Integrations.Deployment, "DEPLOYMENT")
		for k, v := range envVars {
			newEnvVar[k] = v
		}
	}

	return newEnvVar, nil
}

func computeIntegrationConfigToEnvVar(config sdk.JobIntegrationsContext, prefix string) map[string]string {
	envVars := make(map[string]string)
	for k, v := range config.Config {
		suffix := strings.Replace(k, "-", "_", -1)
		suffix = strings.Replace(suffix, ".", "_", -1)
		key := fmt.Sprintf("CDS_INTEGRATION_%s_%s", prefix, suffix)
		envVars[strings.ToUpper(key)] = sdk.OneLineValue(v)
	}
	// integration name
	key := fmt.Sprintf("CDS_INTEGRATION_%s", prefix)
	envVars[strings.ToUpper(key)] = config.Name
	return envVars
}

func (w *CurrentWorker) executeHooksSetupV2(ctx context.Context, fs afero.Fs) error {
	if strings.EqualFold(runtime.GOOS, "windows") {
		log.Warn(ctx, "hooks are not supported on windows")
		return nil
	}
	if len(w.hooks) == 0 {
		return nil
	}

	// Load integrations
	integrationEnv := make([]string, 0)
	if w.currentJobV2.runJobContext.Integrations != nil {
		if w.currentJobV2.runJobContext.Integrations.ArtifactManager.Name != "" {
			envvars := computeIntegrationConfigToEnvVar(w.currentJobV2.runJobContext.Integrations.ArtifactManager, "ARTIFACT_MANAGER")
			for k, v := range envvars {
				integrationEnv = append(integrationEnv, fmt.Sprintf("%s=%s", k, v))
			}
		}
		if w.currentJobV2.runJobContext.Integrations.Deployment.Name != "" {
			envvars := computeIntegrationConfigToEnvVar(w.currentJobV2.runJobContext.Integrations.Deployment, "DEPLOYMENT")
			for k, v := range envvars {
				integrationEnv = append(integrationEnv, fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	var result = make(map[string]string)

	basedir, ok := fs.(*afero.BasePathFs)
	if !ok {
		return sdk.WithStack(fmt.Errorf("invalid given basedir"))
	}

	workerEnv := w.getEnvironmentForWorkerHook()

	for _, h := range w.hooks {
		filepath, err := basedir.RealPath(h.SetupPath)
		if err != nil {
			return sdk.WrapError(err, "cannot get real path for: %s", h.SetupPath)
		}

		msg := sdk.V2SendJobRunInfo{
			Level:   sdk.WorkflowRunInfoLevelInfo,
			Time:    time.Now(),
			Message: "Running worker hook " + h.Config.Label,
		}
		if err := w.ClientV2().V2QueuePushJobInfo(ctx, w.currentJobV2.runJob.Region, w.currentJobV2.runJob.ID, msg); err != nil {
			return sdk.WrapError(err, "cannot record V2QueuePushJobInfo for job (err spawn): %s", w.currentJobV2.runJob.ID)
		}

		str := fmt.Sprintf("source %s ; echo '<<<ENVIRONMENT>>>' ; env", filepath)
		cmd := exec.Command("bash", "-c", str)
		cmd.Env = append(workerEnv, integrationEnv...)
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
					if !strings.HasPrefix(k, "CDS_") && !sdk.IsInArray(k+"="+v, workerEnv) {
						result[k] = v
					}
				}
			}
		}
	}
	w.currentJobV2.envFromHooks = result
	return nil
}

func (w *CurrentWorker) setupHooksV2(ctx context.Context, currentJob CurrentJobV2, fs afero.Fs, workingDir string) error {
	log.Debug(ctx, "Setup hooks")
	if err := fs.MkdirAll(path.Join(workingDir, "setup"), os.FileMode(0700)); err != nil {
		return errors.WithStack(err)
	}
	if err := fs.MkdirAll(path.Join(workingDir, "teardown"), os.FileMode(0700)); err != nil {
		return errors.WithStack(err)
	}

	// Iterate over the integration given on "takeJob"
	if currentJob.runJobContext.Integrations == nil {
		log.Info(ctx, "no integration available for this job")
		return nil
	}

	for _, integ := range currentJob.runJobContext.Integrations.All() {
		log.Info(ctx, "Getting integration %q hooks for project %q", integ.Name, currentJob.runJob.ProjectKey)
		hook, err := w.clientV2.ProjectIntegrationWorkerHookGet(currentJob.runJob.ProjectKey, integ.Name)
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			log.Info(ctx, "no hook found for integration %q", integ.Name)
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

			hookFilename := fmt.Sprintf("%d-%s-%s", hookConfig.Priority, integ.Name, slug.Convert(hookConfig.Label))

			w.hooks = append(w.hooks, workerHook{
				Config:       hookConfig,
				SetupPath:    path.Join(workingDir, "setup", hookFilename),
				TeardownPath: path.Join(workingDir, "teardown", hookFilename),
			})
		}
	}

	for _, h := range w.hooks {
		info := sdk.V2SendJobRunInfo{
			Message: fmt.Sprintf("Setting up worker hook %q", h.Config.Label),
			Level:   sdk.WorkflowRunInfoLevelInfo,
			Time:    time.Now(),
		}

		if err := w.ClientV2().V2QueuePushJobInfo(ctx, w.currentJobV2.runJob.Region, w.currentJobV2.runJob.ID, info); err != nil {
			log.Error(ctx, "runJobServiceReadiness> Unable to send spawn info: %v", err)
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
	return nil
}
