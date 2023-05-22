package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/internal/plugin"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func (w *CurrentWorker) runAsCodeAction(ctx context.Context, currentActionContext sdk.ActionContext, currentActionName string, prefixStepName string) sdk.Result {
	currentAsCodeAction, has := w.currentJob.ascodeAction[currentActionName]
	if !has {
		return w.failAction(ctx, fmt.Sprintf("unknown ascode action: %s", currentActionName))
	}

	// Fill missing input with default values
	for k, v := range currentAsCodeAction.Inputs {
		if _, has := currentActionContext.Inputs[k]; !has {
			currentActionContext.Inputs[k] = v.Default
		}
	}
	// Remove inputs that are not in action definition
	for k := range currentActionContext.Inputs {
		if _, has := currentAsCodeAction.Inputs[k]; !has {
			delete(currentActionContext.Inputs, k)
		}
	}

	for _, step := range currentAsCodeAction.Runs.Steps {
		var result sdk.Result
		switch {
		case step.Uses != "":
			result = w.runAsCodeSubAction(ctx, currentActionContext, step, prefixStepName)
		case step.Run != "":
			result = w.runAsCodeScriptAction(ctx, currentActionContext, step, prefixStepName)
		default:
			return w.failAction(ctx, "invalid action definition. Missing uses or run keys")
		}
		if result.Status == sdk.StatusFail {
			return result
		}
	}
	return sdk.Result{
		Status: sdk.StatusSuccess,
	}
}

func (w *CurrentWorker) runAsCodeScriptAction(ctx context.Context, currentActionContext sdk.ActionContext, step sdk.ActionStep, prefixStepName string) sdk.Result {
	stepName := prefixStepName + "script"
	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step \"%s\"", stepName))

	defer func() {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step \"%s\"", stepName))
	}()

	bts, err := json.Marshal(currentActionContext)
	if err != nil {
		return w.failAction(ctx, fmt.Sprintf("unable to marshal contexts: %v", err))
	}
	var mapContexts map[string]interface{}
	if err := json.Unmarshal(bts, &mapContexts); err != nil {
		return w.failAction(ctx, fmt.Sprintf("unable to unmarshal contexts: %v", err))
	}

	interpolatedInput, err := w.interpolateActionInput(ctx, mapContexts, step.Run)
	if err != nil {
		return w.failAction(ctx, fmt.Sprintf("unable to interpolate script content: %v", err))
	}
	contentString, ok := interpolatedInput.(string)
	if !ok {
		return w.failAction(ctx, fmt.Sprintf("interpolated script content is not a string. Got %T", interpolatedInput))
	}
	result := w.runPlugin(ctx, "script", map[string]string{
		"content": contentString,
	})
	return result
}

func (w *CurrentWorker) runAsCodeSubAction(ctx context.Context, currentActionContext sdk.ActionContext, step sdk.ActionStep, prefixStepName string) sdk.Result {
	actionName := strings.TrimPrefix(step.Uses, "actions/")
	actionPath := strings.Split(actionName, "/")
	stepName := prefixStepName + filepath.Base(actionName)

	w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step \"%s\"", stepName))
	defer func() {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step \"%s\"", stepName))
	}()

	subActionContext, err := w.createSubActionContext(ctx, currentActionContext, step)
	if err != nil {
		return w.failAction(ctx, err.Error())
	}

	switch len(actionPath) {
	case 5:
		// sub action
		return w.runAsCodeAction(ctx, subActionContext, actionName, filepath.Base(actionName)+"/")
	case 1:
		// plugin
		opts := make(map[string]string, 0)
		for k, v := range subActionContext.Inputs {
			vString, ok := v.(string)
			if !ok {
				return w.failAction(ctx, fmt.Sprintf("input %s is not a string. Got %T", k, v))
			}
			opts[k] = vString
		}
		return w.runPlugin(ctx, actionPath[0], opts)
	default:
		return w.failAction(ctx, fmt.Sprintf("Unknown step %s", actionName))
	}
}

func (w *CurrentWorker) createSubActionContext(ctx context.Context, currentActionContext sdk.ActionContext, step sdk.ActionStep) (sdk.ActionContext, error) {
	// Interpolate subaction inputs filled by parent
	bts, err := json.Marshal(currentActionContext)
	if err != nil {
		return sdk.ActionContext{}, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to marshal current context")
	}
	var mapContexts map[string]interface{}
	if err := json.Unmarshal(bts, &mapContexts); err != nil {
		return sdk.ActionContext{}, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to unmarshal current context")
	}
	subActionContext := sdk.ActionContext{
		Inputs: make(map[string]interface{}),
	}

	for k, v := range step.With {
		interpolatedInput, err := w.interpolateActionInput(ctx, mapContexts, v)
		if err != nil {
			return sdk.ActionContext{}, sdk.NewErrorFrom(sdk.ErrInvalidData, fmt.Sprintf("unable to interpolate input %s: %v", k, err))
		}
		subActionContext.Inputs[k] = interpolatedInput
	}
	return subActionContext, nil
}

func (w *CurrentWorker) failAction(ctx context.Context, reason string) sdk.Result {
	res := sdk.Result{
		Status: sdk.StatusFail,
		Reason: reason,
	}
	log.Error(ctx, "worker.runAsCodeAction> %v", res.Reason)
	w.SendLog(ctx, workerruntime.LevelError, res.Reason)
	return res
}

func (w *CurrentWorker) runPlugin(ctx context.Context, pluginName string, opts map[string]string) sdk.Result {
	pluginClient, err := plugin.NewClient(ctx, w, plugin.TypeAction, pluginName)
	if pluginClient != nil {
		defer pluginClient.Close(ctx)
	}
	if err != nil {
		return w.failAction(ctx, fmt.Sprintf("%v", err))
	}

	res := pluginClient.Run(ctx, opts)
	if err != nil {
		return w.failAction(ctx, fmt.Sprintf("error runnning artifact %s: %v", pluginName, err))
	}
	if res.Status == sdk.StatusFail {
		return w.failAction(ctx, res.Details)
	}
	return sdk.Result{Status: res.Status}
}

func (w *CurrentWorker) interpolateActionInput(ctx context.Context, contexts map[string]interface{}, input string) (interface{}, error) {
	ap := sdk.NewActionParser(contexts, sdk.DefaultFuncs)
	interpolatedInput, err := ap.Interpolate(ctx, input)
	if err != nil {
		return "", err
	}
	return interpolatedInput, nil
}
