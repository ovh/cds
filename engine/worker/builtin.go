package main

import (
	"fmt"
	"os"
	"path"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/plugin"
)

func runBuiltin(a *sdk.Action, pbJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {
	res := sdk.Result{Status: sdk.StatusFail}
	switch a.Name {
	case sdk.ArtifactUpload:
		filePattern, tag := getArtifactParams(a)
		return runArtifactUpload(filePattern, tag, pbJob, stepOrder)
	case sdk.ArtifactDownload:
		return runArtifactDownload(a, pbJob, stepOrder)
	case sdk.ScriptAction:
		return runScriptAction(a, pbJob, stepOrder)
	case sdk.JUnitAction:
		return runParseJunitTestResultAction(a, pbJob, stepOrder)
	case sdk.GitCloneAction:
		return runGitClone(a, pbJob, stepOrder)
	}

	res.Reason = fmt.Sprintf("Unknown builtin step: %s\n", name)
	return res
}

func runPlugin(a *sdk.Action, pbJob sdk.PipelineBuildJob, stepOrder int) sdk.Result {
	res := sdk.Result{Status: sdk.StatusFail}
	//For the moment we consider that plugin name = action name = plugin binary file name
	pluginName := a.Name
	//The binary file has been downloaded during requirement check in /tmp
	pluginBinary := path.Join(os.TempDir(), a.Name)

	var tlsskipverify bool
	if os.Getenv("CDS_SKIP_VERIFY") != "" {
		tlsskipverify = true
	}

	//Create the rpc server
	pluginClient := plugin.NewClient(pluginName, pluginBinary, WorkerID, api, tlsskipverify)
	defer pluginClient.Kill()

	//Get the plugin interface
	_plugin, err := pluginClient.Instance()
	if err != nil {
		result := sdk.Result{
			Status: sdk.StatusFail,
			Reason: fmt.Sprintf("Unable to init plugin %s: %s\n", pluginName, err),
		}
		sendLog(pbJob.ID, result.Reason, pbJob.PipelineBuildID, stepOrder, false)
		return result
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
		res.Status = sdk.StatusSuccess
	}
	return res
}
