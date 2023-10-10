package action

import (
	"context"
	"errors"
	"fmt"

	"github.com/ovh/cds/engine/worker/internal/plugin"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func RunPromote(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, _ []sdk.Variable) (sdk.Result, error) {
	jobID, err := workerruntime.JobID(ctx)
	if err != nil {
		return sdk.Result{Status: sdk.StatusFail}, err
	}

	promotedRunResultIDs, err := RunReleaseActionPrepare(ctx, wk, a)
	if err != nil {
		return sdk.Result{Status: sdk.StatusFail}, err
	}

	if sdk.ParameterValue(a.Parameters, "srcMaturity") != "" {
		wk.SendLog(ctx, workerruntime.LevelInfo, "# Param: \"srcMaturity\" is deprecated, value is ignored")
	}

	log.Info(ctx, "RunPromote> preparing run result %+v for promotion", promotedRunResultIDs)
	wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Preparing run results %v for promotion to %q", promotedRunResultIDs, sdk.ParameterValue(a.Parameters, "destMaturity")))
	if err := wk.Client().QueueWorkflowRunResultsPromote(ctx,
		jobID, promotedRunResultIDs,
		sdk.ParameterValue(a.Parameters, "destMaturity"),
	); err != nil {
		return sdk.Result{Status: sdk.StatusFail}, err
	}

	pfName := sdk.ParameterFind(wk.Parameters(), "cds.integration.artifact_manager")
	if pfName == nil {
		return sdk.Result{}, errors.New("unable to retrieve artifact manager integration... Aborting")
	}

	pluginClient, err := plugin.NewClient(ctx, wk, plugin.TypeIntegration, sdk.GRPCPluginPromote, plugin.InputManagementDefault)
	if err != nil {
		return sdk.Result{Status: sdk.StatusFail, Reason: fmt.Sprintf("unable to start GRPCPlugin: %v", err)}, nil
	}
	defer pluginClient.Close(ctx)

	opts := sdk.ParametersToMap(wk.Parameters())
	for _, v := range a.Parameters {
		opts[v.Name] = v.Value
	}

	res := pluginClient.Run(ctx, opts)

	return sdk.Result{
		Status: res.Status,
		Reason: res.Details,
	}, nil
}
