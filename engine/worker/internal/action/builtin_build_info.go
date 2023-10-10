package action

import (
	"context"
	"errors"
	"fmt"
	"github.com/ovh/cds/engine/worker/internal/plugin"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func PushBuildInfo(ctx context.Context, wk workerruntime.Runtime, _ sdk.Action, _ []sdk.Variable) (sdk.Result, error) {
	pfName := sdk.ParameterFind(wk.Parameters(), "cds.integration.artifact_manager")
	if pfName == nil {
		return sdk.Result{}, errors.New("unable to retrieve artifact manager integration... Aborting")
	}

	pluginClient, err := plugin.NewClient(ctx, wk, plugin.TypeIntegration, sdk.GRPCPluginBuildInfo, plugin.InputManagementDefault)
	if err != nil {
		return sdk.Result{Status: sdk.StatusFail, Reason: fmt.Sprintf("unable to start GRPCPlugin: %v", err)}, nil
	}
	defer pluginClient.Close(ctx)

	res := pluginClient.Run(ctx, sdk.ParametersToMap(wk.Parameters()))

	return sdk.Result{
		Status: res.Status,
		Reason: res.Details,
	}, nil
}
