package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
	"github.com/ovh/cds/sdk/log"
)

func runDeployApplication(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, secrets []sdk.Variable, sendLog LoggerFunc) sdk.Result {
		pfName := sdk.ParameterFind(params, "cds.integration")
		if pfName == nil {
			res := sdk.Result{
				Reason: "Unable to retrieve deployment integration... Aborting",
				Status: sdk.StatusFail,
			}
			sendLog(res.Reason)
			return res
		}

		pkey := sdk.ParameterFind(params, "cds.project")
		pf, err := w.client.ProjectIntegrationGet(pkey.Value, pfName.Value, true)
		if err != nil {
			res := sdk.Result{
				Reason: fmt.Sprintf("Unable to retrieve deployment integration (%v)... Aborting", err),
				Status: sdk.StatusFail,
			}
			sendLog(res.Reason)
			return res
		}

		//First check OS and Architecture
		var currentOS = strings.ToLower(sdk.GOOS)
		var currentARCH = strings.ToLower(sdk.GOARCH)
		var binary *sdk.GRPCPluginBinary
		for _, b := range w.currentJob.wJob.IntegrationPluginBinaries {
			if b.OS == currentOS && b.Arch == currentARCH {
				binary = &b
				break
			}
		}

		if binary == nil {
			res := sdk.Result{
				Reason: fmt.Sprintf("Unable to retrieve the plugin for deployment integration %s... Aborting", pf.Model.Name),
				Status: sdk.StatusFail,
			}
			sendLog(res.Reason)
			return res
		}

		pluginSocket, err := startGRPCPlugin(context.Background(), binary.PluginName, w, binary, startGRPCPluginOptions{})
		if err != nil {
			res := sdk.Result{
				Reason: "Unable to startGRPCPlugin... Aborting",
				Status: sdk.StatusFail,
			}
			sendLog(err.Error())
			return res
		}

		c, err := integrationplugin.Client(context.Background(), pluginSocket.Socket)
		if err != nil {
			res := sdk.Result{
				Reason: "Unable to call grpc plugin... Aborting",
				Status: sdk.StatusFail,
			}
			sendLog(err.Error())
			return res
		}

		pluginSocket.Client = c
		if _, err := c.Manifest(context.Background(), new(empty.Empty)); err != nil {
			res := sdk.Result{
				Reason: "Unable to call grpc plugin manifest... Aborting",
				Status: sdk.StatusFail,
			}
			sendLog(err.Error())
			return res
		}

		pluginClient := pluginSocket.Client
		integrationPluginClient, ok := pluginClient.(integrationplugin.IntegrationPluginClient)
		if !ok {
			res := sdk.Result{
				Reason: "Unable to retrieve plugin client... Aborting",
				Status: sdk.StatusFail,
			}
			sendLog(res.Reason)
			return res
		}

		logCtx, stopLogs := context.WithCancel(ctx)
		done := make(chan struct{})
		go enablePluginLogger(logCtx, done, sendLog, pluginSocket)

		manifest, err := integrationPluginClient.Manifest(ctx, &empty.Empty{})
		if err != nil {
			res := sdk.Result{
				Reason: "Unable to retrieve plugin manifest... Aborting",
				Status: sdk.StatusFail,
			}
			integrationPluginClientStop(ctx, integrationPluginClient, done, stopLogs)
			sendLog(err.Error())
			return res
		}

		sendLog(fmt.Sprintf("# Plugin %s v%s is ready", manifest.Name, manifest.Version))

		query := integrationplugin.DeployQuery{
			Options: sdk.ParametersToMap(*params),
		}

		res, err := integrationPluginClient.Deploy(ctx, &query)
		if err != nil {
			res := sdk.Result{
				Reason: fmt.Sprintf("Error deploying application: %v", err),
				Status: sdk.StatusFail,
			}
			integrationPluginClientStop(ctx, integrationPluginClient, done, stopLogs)
			return res
		}

		sendLog(fmt.Sprintf("# Details: %s", res.Details))
		sendLog(fmt.Sprintf("# Status: %s", res.Status))

		if strings.ToUpper(res.Status) == strings.ToUpper(sdk.StatusSuccess) {
			integrationPluginClientStop(ctx, integrationPluginClient, done, stopLogs)
			return sdk.Result{
				Status: sdk.StatusSuccess,
			}
		}

		integrationPluginClientStop(ctx, integrationPluginClient, done, stopLogs)

		return sdk.Result{
			Status: sdk.StatusFail,
			Reason: res.Details,
		}
	}
}

func integrationPluginClientStop(ctx context.Context, integrationPluginClient integrationplugin.IntegrationPluginClient, done chan struct{}, stopLogs context.CancelFunc) {
	if _, err := integrationPluginClient.Stop(ctx, new(empty.Empty)); err != nil {
		// Transport is closing is a "normal" error, as we requested plugin to stop
		if !strings.Contains(err.Error(), "transport is closing") {
			log.Error("Error on integrationPluginClient.Stop: %s", err)
		}
	}
	stopLogs()
	<-done
}
