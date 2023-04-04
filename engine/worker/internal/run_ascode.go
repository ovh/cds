package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/internal/action"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func (w *CurrentWorker) runAscodeAction(ctx context.Context, actionName string) sdk.Result {
	// TODO manage plugins - inputs (interpolation with context)

	currentAscodeAction, has := w.currentJob.ascodeAction[actionName]
	if !has {
		return w.failAction(ctx, fmt.Sprintf("unknown ascode action: %s", actionName))
	}

	for _, step := range currentAscodeAction.Runs.Steps {
		switch {
		case step.Uses != "":
			log.Debug(ctx, "running subaction %s", step.Uses)

			res := w.runAscodeAction(ctx, strings.TrimPrefix(step.Uses, "actions/"))
			if res.Status == sdk.StatusFail {
				return res
			}

		case step.Run != "":
			log.Debug(ctx, "running script action")

			//TODO replace by script plugin
			fakeAction := sdk.Action{
				Parameters: []sdk.Parameter{
					{
						Name:  "script",
						Value: step.Run,
					},
				},
			}
			res, err := action.RunScriptAction(ctx, w, fakeAction, nil)
			if err != nil {
				return w.failAction(ctx, fmt.Sprintf("unable to execute script: %v", err))
			}
			if res.Status == sdk.StatusFail {
				return res
			}

		}
	}
	return sdk.Result{
		Status: sdk.StatusSuccess,
	}
}

func (w *CurrentWorker) failAction(ctx context.Context, reason string) sdk.Result {
	res := sdk.Result{
		Status: sdk.StatusFail,
		Reason: reason,
	}
	log.Error(ctx, "worker.runAscodeAction> %v", res.Reason)
	w.SendLog(ctx, workerruntime.LevelError, res.Reason)
	return res
}
