package main

import (
	"fmt"
	"os"
	"path"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/plugin"
)

func runBuiltin(a *sdk.Action, actionBuild sdk.ActionBuild) sdk.Result {
	res := sdk.Result{Status: sdk.StatusFail}
	switch a.Name {
	case sdk.ArtifactUpload:
		filePattern, tag := getArtifactParams(a)
		return runArtifactUpload(filePattern, tag, actionBuild)
	case sdk.ArtifactDownload:
		return runArtifactDownload(a, actionBuild)
	case sdk.ScriptAction:
		return runScriptAction(a, actionBuild)
	case sdk.NotifAction:
		return runNotifAction(a, actionBuild)
	case sdk.JUnitAction:
		return runParseJunitTestResultAction(a, actionBuild)
	}

	sendLog(actionBuild.ID, name, fmt.Sprintf("Unknown builtin step: %s\n", name))
	return res
}

func runPlugin(a *sdk.Action, actionBuild sdk.ActionBuild) sdk.Result {
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
		sendLog(actionBuild.ID, "PLUGIN", fmt.Sprintf("Unable to init plugin %s: %s\n", pluginName, err))
		return sdk.Result{Status: sdk.StatusFail}
	}

	//Manage all parameters
	pluginArgs := plugin.Arguments{
		Data: map[string]string{},
	}
	for _, p := range a.Parameters {
		pluginArgs.Data[p.Name] = p.Value
	}
	for _, p := range actionBuild.Args {
		pluginArgs.Data[p.Name] = p.Value
	}

	//Call the Run function on the plugin interface
	pluginAction := plugin.Action{
		IDActionBuild: actionBuild.ID,
		Args:          pluginArgs,
	}

	sendLog(actionBuild.ID, "PLUGIN", fmt.Sprintf("Starting plugin: %s\n", pluginName))
	pluginResult := _plugin.Run(pluginAction)
	sendLog(actionBuild.ID, "PLUGIN", fmt.Sprintf("Plugin %s finished with status: %s\n", pluginName, pluginResult))

	if pluginResult == plugin.Success {
		res.Status = sdk.StatusSuccess
	}
	return res
}
