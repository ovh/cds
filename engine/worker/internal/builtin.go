package internal

import (
	"context"
	"fmt"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/internal/action"
	"github.com/ovh/cds/engine/worker/internal/plugin"
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

	pluginClient, err := plugin.NewClient(ctx, w, plugin.TypeAction, a.Name, plugin.InputManagementDefault)
	if err != nil {
		return sdk.Result{Status: sdk.StatusFail, Reason: fmt.Sprintf("Unable to start grpc plugin... Aborting (%v)", err)}
	}
	defer pluginClient.Close(ctx)

	opts := sdk.ParametersMapMerge(sdk.ParametersToMap(w.currentJob.params), sdk.ParametersToMap(a.Parameters), sdk.MapMergeOptions.ExcludeGitParams)

	result := pluginClient.Run(ctx, opts)
	return sdk.Result{Status: result.Status, Reason: result.Details}
}
