package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/platformplugin"
)

func runDeployApplication(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		pfName := sdk.ParameterFind(params, "cds.platform")
		if pfName == nil {
			res := sdk.Result{
				Reason: "Unable to retrieve deployment platform... Aborting",
				Status: sdk.StatusFail.String(),
			}
			sendLog(res.Reason)
			return res
		}

		pkey := sdk.ParameterFind(params, "cds.project")
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

		manifest, err := platformPluginClient.Manifest(ctx, &empty.Empty{})
		if err != nil {
			res := sdk.Result{
				Reason: "Unable to retrieve plugin manifest... Aborting",
				Status: sdk.StatusFail.String(),
			}
			sendLog(err.Error())
			return res
		}

		sendLog(fmt.Sprintf("# Plugin %s v%s is ready", manifest.Name, manifest.Version))

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

		sendLog(fmt.Sprintf("# Details: %s", res.Details))
		sendLog(fmt.Sprintf("# Status: %s", res.Status))

		if strings.ToUpper(res.Status) == strings.ToUpper(sdk.StatusSuccess.String()) {
			return sdk.Result{
				Status: sdk.StatusSuccess.String(),
			}
		}

		return sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: res.Details,
		}
	}
}
