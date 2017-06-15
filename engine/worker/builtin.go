package main

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/plugin"
)

func (w *currentWorker) runBuiltin(ctx context.Context, a *sdk.Action, pbJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {
	defer w.drainLogsAndCloseLogger(ctx)

	res := sdk.Result{Status: sdk.StatusFail.String()}
	switch a.Name {
	case sdk.ArtifactUpload:
		filePattern, tag := getArtifactParams(a)
		return w.runArtifactUpload(ctx, filePattern, tag, pbJob, stepOrder)
	case sdk.ArtifactDownload:
		return w.runArtifactDownload(ctx, a, pbJob, stepOrder)
	case sdk.ScriptAction:
		return w.runScriptAction(ctx, a, pbJob, stepOrder)
	case sdk.JUnitAction:
		return w.runParseJunitTestResultAction(ctx, a, pbJob, stepOrder)
	case sdk.GitCloneAction:
		return w.runGitClone(ctx, a, pbJob, stepOrder)
	}
	res.Reason = fmt.Sprintf("Unknown builtin step: %s\n", a.Name)
	return res
}

func (w *currentWorker) runPlugin(ctx context.Context, a *sdk.Action, pbJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {

	chanRes := make(chan sdk.Result)

	go func(pbJob *sdk.PipelineBuildJob) {
		res := sdk.Result{Status: sdk.StatusFail.String()}

		//For the moment we consider that plugin name = action name = plugin binary file name
		pluginName := a.Name
		//The binary file has been downloaded during requirement check in /tmp
		pluginBinary := path.Join(os.TempDir(), a.Name)

		var tlsskipverify bool
		if os.Getenv("CDS_SKIP_VERIFY") != "" {
			tlsskipverify = true
		}

		//TODO: cancel the plugin

		//Create the rpc server
		pluginClient := plugin.NewClient(pluginName, pluginBinary, w.id, w.apiEndpoint, tlsskipverify)
		defer pluginClient.Kill()

		//Get the plugin interface
		_plugin, err := pluginClient.Instance()
		if err != nil {
			result := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to init plugin %s: %s\n", pluginName, err),
			}
			w.sendLog(pbJob.ID, result.Reason, pbJob.PipelineBuildID, stepOrder, false)
			chanRes <- result
		}

		//Manage all parameters
		pluginArgs := plugin.Arguments{
			Data: map[string]string{},
		}
		for _, p := range a.Parameters {
			pluginArgs.Data[p.Name] = p.Value
		}
		for _, p := range pbJob.Parameters {
			pluginArgs.Data[p.Name] = p.Value
		}

		//Call the Run function on the plugin interface
		pluginAction := plugin.Job{
			IDPipelineBuild:    pbJob.PipelineBuildID,
			IDPipelineJobBuild: pbJob.ID,
			OrderStep:          stepOrder,
			Args:               pluginArgs,
		}

		pluginResult := _plugin.Run(pluginAction)

		if pluginResult == plugin.Success {
			res.Status = sdk.StatusSuccess.String()
		}

		chanRes <- res
	}(&pbJob)

	for {
		select {
		case <-ctx.Done():
			log.Error("CDS Worker execution canceled: %v", ctx.Err())
			w.sendLog(pbJob.ID, "CDS Worker execution canceled\n", pbJob.PipelineBuildID, stepOrder, false)
			return sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "CDS Worker execution canceled",
			}

		case res := <-chanRes:
			return res
		}
	}
}
