package main

import (
	"fmt"
	"os"
	"path"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/plugin"
)

func runBuiltin(a *sdk.Action, pbJob sdk.PipelineBuildJob) sdk.Result {
	res := sdk.Result{Status: sdk.StatusFail}
	switch a.Name {
	case sdk.ArtifactUpload:
		filePattern, tag := getArtifactParams(a)
		return runArtifactUpload(filePattern, tag, pbJob)
	case sdk.ArtifactDownload:
		return runArtifactDownload(a, pbJob)
	case sdk.ScriptAction:
		return runScriptAction(a, pbJob)
	case sdk.JUnitAction:
		return runParseJunitTestResultAction(a, pbJob)
	}

	sendLog(pbJob.ID, name, fmt.Sprintf("Unknown builtin step: %s\n", name), pbJob.PipelineBuildID)
	return res
}

func runPlugin(a *sdk.Action, pbJob sdk.PipelineBuildJob) sdk.Result {
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
		sendLog(pbJob.ID, "PLUGIN", fmt.Sprintf("Unable to init plugin %s: %s\n", pluginName, err), pbJob.PipelineBuildID)
		return sdk.Result{Status: sdk.StatusFail}
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
		Args:               pluginArgs,
	}

	sendLog(pbJob.ID, "PLUGIN", fmt.Sprintf("Starting plugin: %s\n", pluginName), pbJob.PipelineBuildID)
	pluginResult := _plugin.Run(pluginAction)
	sendLog(pbJob.ID, "PLUGIN", fmt.Sprintf("Plugin %s finished with status: %s\n", pluginName, pluginResult), pbJob.PipelineBuildID)

	if pluginResult == plugin.Success {
		res.Status = sdk.StatusSuccess
	}
	return res
}
