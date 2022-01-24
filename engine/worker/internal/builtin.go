package internal

import (
	"context"
	"fmt"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/internal/action"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

var mapBuiltinActions = map[string]BuiltInAction{}

func init() {
	mapBuiltinActions[sdk.ArtifactUpload] = action.RunArtifactUpload
	mapBuiltinActions[sdk.ArtifactDownload] = action.RunArtifactDownload
	mapBuiltinActions[sdk.ScriptAction] = action.RunScriptAction
	mapBuiltinActions[sdk.JUnitAction] = action.RunParseJunitTestResultAction
	mapBuiltinActions[sdk.GitCloneAction] = action.RunGitClone
	mapBuiltinActions[sdk.GitTagAction] = action.RunGitTag
	mapBuiltinActions[sdk.ReleaseVCSAction] = action.RunReleaseVCS
	mapBuiltinActions[sdk.ReleaseAction] = action.RunRelease
	mapBuiltinActions[sdk.PromoteAction] = action.RunPromote
	mapBuiltinActions[sdk.CheckoutApplicationAction] = action.RunCheckoutApplication
	mapBuiltinActions[sdk.DeployApplicationAction] = action.RunDeployApplication
	mapBuiltinActions[sdk.CoverageAction] = action.RunParseCoverageResultAction
	mapBuiltinActions[sdk.InstallKeyAction] = action.RunInstallKey
	mapBuiltinActions[sdk.PushBuildInfo] = action.PushBuildInfo
}

func (w *CurrentWorker) runBuiltin(ctx context.Context, a sdk.Action, secrets []sdk.Variable) sdk.Result {
	f, ok := mapBuiltinActions[a.Name]
	if !ok {
		res := sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("unknown builtin step: %s", a.Name),
		}
		log.Error(ctx, "worker.runBuiltin> %v", res.Reason)
		w.SendLog(ctx, workerruntime.LevelError, res.Reason)
		return res
	}

	log.Debug(ctx, "running builin action %s %s", a.StepName, a.Name)
	res, err := f(ctx, w, a, secrets)
	if err != nil {
		res.Status = sdk.StatusFail
		res.Reason = err.Error()
		log.Error(ctx, "worker.runBuiltin> %v", err)
		w.SendLog(ctx, workerruntime.LevelError, res.Reason)
	}
	return res
}

func (w *CurrentWorker) runGRPCPlugin(ctx context.Context, a sdk.Action) sdk.Result {
	log.Info(ctx, "running grpc plugin %q", a.Name)

	chanRes := make(chan sdk.Result, 1)
	done := make(chan struct{})
	sdk.NewGoRoutines(ctx).Run(ctx, "runGRPCPlugin", func(ctx context.Context) {
		action.RunGRPCPlugin(ctx, a.Name, w.currentJob.params, a, w, chanRes, done)
	})

	select {
	case <-ctx.Done():
		log.Error(ctx, "CDS Worker execution cancelled: %v", ctx.Err())
		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: "CDS Worker execution cancelled",
		}
	case res := <-chanRes:
		// Useful to wait all logs are send before sending final status and log
		<-done
		return res
	}
}
