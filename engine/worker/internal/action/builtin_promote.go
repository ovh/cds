package action

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
)

func RunPromote(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, _ []sdk.Variable) (sdk.Result, error) {
	pfName := sdk.ParameterFind(wk.Parameters(), "cds.integration.artifact_manager")
	if pfName == nil {
		return sdk.Result{}, errors.New("unable to retrieve artifact manager integration... Aborting")
	}

	plugin := wk.GetPlugin(sdk.GRPCPluginPromote)
	if plugin == nil {
		return sdk.Result{}, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find plugin of type %s", sdk.GRPCPluginPromote)
	}

	//First check OS and Architecture
	binary := plugin.GetBinary(strings.ToLower(sdk.GOOS), strings.ToLower(sdk.GOARCH))
	if binary == nil {
		return sdk.Result{}, fmt.Errorf("unable to retrieve the plugin for promote on integration %s... Aborting", pfName.Value)
	}

	pluginSocket, err := startGRPCPlugin(ctx, binary.PluginName, wk, binary, startGRPCPluginOptions{})
	if err != nil {
		return sdk.Result{}, fmt.Errorf("unable to start GRPCPlugin: %v", err)
	}

	c, err := integrationplugin.Client(context.Background(), pluginSocket.Socket)
	if err != nil {
		return sdk.Result{}, fmt.Errorf("unable to call GRPCPlugin: %v", err)
	}

	qPort := integrationplugin.WorkerHTTPPortQuery{Port: wk.HTTPPort()}
	if _, err := c.WorkerHTTPPort(ctx, &qPort); err != nil {
		return sdk.Result{}, fmt.Errorf("unable to setup plugin with worker port: %v", err)
	}

	pluginSocket.Client = c
	if _, err := c.Manifest(context.Background(), new(empty.Empty)); err != nil {
		return sdk.Result{}, fmt.Errorf("unable to call GRPCPlugin: %v", err)
	}

	pluginClient := pluginSocket.Client
	integrationPluginClient, ok := pluginClient.(integrationplugin.IntegrationPluginClient)
	if !ok {
		return sdk.Result{}, fmt.Errorf("unable to retrieve integration GRPCPlugin: %v", err)
	}

	logCtx, stopLogs := context.WithCancel(ctx)
	done := make(chan struct{})
	go enablePluginLogger(logCtx, done, pluginSocket, wk)

	defer integrationPluginClientStop(ctx, integrationPluginClient, done, stopLogs)

	manifest, err := integrationPluginClient.Manifest(ctx, &empty.Empty{})
	if err != nil {
		integrationPluginClientStop(ctx, integrationPluginClient, done, stopLogs)
		return sdk.Result{}, fmt.Errorf("unable to retrieve retrieve plugin manifest: %v", err)
	}

	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Plugin %s v%s is ready", manifest.Name, manifest.Version))

	query := integrationplugin.RunQuery{
		Options: sdk.ParametersToMap(wk.Parameters()),
	}
	for _, v := range a.Parameters {
		query.Options[v.Name] = v.Value
	}

	res, err := integrationPluginClient.Run(ctx, &query)
	if err != nil {
		integrationPluginClientStop(ctx, integrationPluginClient, done, stopLogs)
		return sdk.Result{}, fmt.Errorf("error while running integration plugin: %v", err)
	}

	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Details: %s", res.Details))
	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("# Status: %s", res.Status))

	if strings.EqualFold(res.Status, sdk.StatusSuccess) {
		integrationPluginClientStop(ctx, integrationPluginClient, done, stopLogs)
		return sdk.Result{
			Status: sdk.StatusSuccess,
		}, nil
	}

	return sdk.Result{
		Status: sdk.StatusFail,
		Reason: res.Details,
	}, nil
}
