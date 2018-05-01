package main

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk/grpcplugin/platformplugin"

	"github.com/ovh/cds/sdk"
)

func runDeployApplication(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		sendLog("# Starting application deployment...")

		pkey := sdk.ParameterFind(params, "cds.project")

		pfName := sdk.ParameterFind(params, "cds.platform")
		if pfName == nil {
			res := sdk.Result{
				Reason: "Unable to retrieve deployment platform... Aborting",
				Status: sdk.StatusFail.String(),
			}
			sendLog(res.Reason)
			return res
		}

		pf, err := w.client.ProjectPlatform(pkey.Value, pfName.Value, true)
		if err != nil {
			res := sdk.Result{
				Reason: fmt.Sprintf("Unable to retrieve deployment platform (%v)... Aborting", err),
				Status: sdk.StatusFail.String(),
			}
			sendLog(res.Reason)
			return res
		}

		pluginSocket, has := w.mapPluginClient[pf.Model.PluginName]
		if !has {
			res := sdk.Result{
				Reason: "Unable to retrieve plugin client... Aborting",
				Status: sdk.StatusFail.String(),
			}
			sendLog(res.Reason)
			return res
		}

		pluginClient := pluginSocket.Client
		platformPluginClient, ok := pluginClient.(platformplugin.PlatformPluginClient)
		if !ok {
			res := sdk.Result{
				Reason: "Unable to retrieve plugin client... Aborting",
				Status: sdk.StatusFail.String(),
			}
			sendLog(res.Reason)
			return res
		}

		logCtx, stopLogs := context.WithCancel(ctx)
		go enablePluginLogger(logCtx, sendLog, pluginSocket)
		defer stopLogs()

		sendLog(fmt.Sprintf("# Plugin %s is ready", pf.Model.PluginName))

		query := platformplugin.DeployQuery{
			Options: sdk.ParametersToMap(*params),
		}

		res, err := platformPluginClient.Deploy(ctx, &query)
		if err != nil {
			res := sdk.Result{
				Reason: fmt.Sprintf("Error deploying application: %v", err),
				Status: sdk.StatusFail.String(),
			}
			sendLog(res.Reason)
			return res
		}

		sendLog("# Deploy successfully called")

		sendLog(fmt.Sprintf("# Details: %s", res.Details))
		sendLog(fmt.Sprintf("# Status: %s", res.Status))

		return sdk.Result{
			Status: sdk.StatusSuccess.String(),
		}
	}
}
