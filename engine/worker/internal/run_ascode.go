package internal

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/internal/plugin"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func (w *CurrentWorker) runAscodeAction(ctx context.Context, actionName string, prefixStepName string) sdk.Result {
	currentAscodeAction, has := w.currentJob.ascodeAction[actionName]
	if !has {
		return w.failAction(ctx, fmt.Sprintf("unknown ascode action: %s", actionName))
	}

	for _, step := range currentAscodeAction.Runs.Steps {
		switch {
		case step.Uses != "":
			actionName := strings.TrimPrefix(step.Uses, "actions/")
			actionPath := strings.Split(actionName, "/")
			stepName := prefixStepName + filepath.Base(actionName)

			w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step \"%s\"", stepName))

			switch len(actionPath) {
			case 5:
				// sub action
				res := w.runAscodeAction(ctx, actionName, filepath.Base(actionName)+"/")
				if res.Status == sdk.StatusFail {
					return res
				}
			case 1:
				// plugin
				res := w.runPlugin(ctx, actionPath[0], step.With)
				if res.Status == sdk.StatusFail {
					return res
				}
			default:
				msg := fmt.Sprintf("Unknown step %s", actionName)
				w.SendLog(ctx, workerruntime.LevelError, msg)
				return sdk.Result{Status: sdk.StatusFail, Reason: msg}
			}

			w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step \"%s\"", stepName))

		case step.Run != "":
			stepName := prefixStepName + "script"

			w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Starting step \"%s\"", stepName))
			res := w.runPlugin(ctx, "script", map[string]string{
				"content": step.Run,
			})
			if res.Status == sdk.StatusFail {
				return res
			}
			w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("End of step \"%s\"", stepName))
		}
	}
	return sdk.Result{
		Status: sdk.StatusSuccess,
	}
}

func (wk *CurrentWorker) failAction(ctx context.Context, reason string) sdk.Result {
	res := sdk.Result{
		Status: sdk.StatusFail,
		Reason: reason,
	}
	log.Error(ctx, "worker.runAscodeAction> %v", res.Reason)
	wk.SendLog(ctx, workerruntime.LevelError, res.Reason)
	return res
}

func (wk *CurrentWorker) runPlugin(ctx context.Context, pluginName string, opts map[string]string) sdk.Result {
	pluginClient, err := plugin.NewClient(ctx, wk, plugin.TypeAction, pluginName)
	if pluginClient != nil {
		defer pluginClient.Close(ctx)
	}
	if err != nil {
		return wk.failAction(ctx, fmt.Sprintf("%v", err))
	}

	res := pluginClient.Run(ctx, opts)
	if err != nil {
		return wk.failAction(ctx, fmt.Sprintf("error uploading artifact: %v", err))
	}
	return sdk.Result{Status: res.Status, Reason: res.Details}
}
