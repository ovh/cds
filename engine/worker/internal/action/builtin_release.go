package action

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
)

func RunReleaseActionPrepare(ctx context.Context, wk workerruntime.Runtime, a sdk.Action) ([]string, error) {
	artifactList := sdk.ParameterValue(a.Parameters, "artifacts")
	artSplitted := strings.Split(artifactList, ",")
	artRegs := make([]*regexp.Regexp, 0, len(artSplitted))
	for _, arti := range artSplitted {
		r, err := regexp.Compile(arti)
		if err != nil {
			return nil, sdk.Errorf("unable to compile regexp in artifact list: %v", err)
		}
		artRegs = append(artRegs, r)
	}

	projectKey := sdk.ParameterValue(wk.Parameters(), "cds.project")
	wName := sdk.ParameterValue(wk.Parameters(), "cds.workflow")
	runNumberString := sdk.ParameterValue(wk.Parameters(), "cds.run.number")
	runNumber, err := strconv.ParseInt(runNumberString, 10, 64)
	if err != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("cannot parse '%s' as run number: %s", runNumberString, err))
		return nil, newError
	}
	runResult, err := wk.Client().WorkflowRunResultsList(ctx, projectKey, wName, runNumber)
	if err != nil {
		return nil, sdk.Errorf("unable to get run result list: %v", err)
	}

	promotedRunResultIDs := make([]string, 0)
	for _, r := range runResult {
		// static-file type does not need to be released
		if r.Type == sdk.WorkflowRunResultTypeStaticFile {
			continue
		}
		rData, err := r.GetArtifactManager()
		if err != nil {
			return nil, sdk.Errorf("unable to read artifacts data: %v", err)
		}
		skip := true
		for _, reg := range artRegs {
			if reg.MatchString(rData.Name) {
				skip = false
				break
			}
		}
		if !skip {
			promotedRunResultIDs = append(promotedRunResultIDs, r.ID)
		}
	}

	return promotedRunResultIDs, nil
}

func RunRelease(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, _ []sdk.Variable) (sdk.Result, error) {
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

	log.Info(ctx, "RunRelease> preparing run result %+v for release", promotedRunResultIDs)
	if err := wk.Client().QueueWorkflowRunResultsRelease(ctx, jobID, promotedRunResultIDs, sdk.ParameterValue(a.Parameters, "destMaturity")); err != nil {
		return sdk.Result{Status: sdk.StatusFail}, err
	}

	pfName := sdk.ParameterFind(wk.Parameters(), "cds.integration.artifact_manager")
	if pfName == nil {
		return sdk.Result{}, errors.New("unable to retrieve artifact manager integration... Aborting")
	}

	plugin := wk.GetPlugin(sdk.GRPCPluginRelease)
	if plugin == nil {
		return sdk.Result{}, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find plugin of type %s", sdk.GRPCPluginRelease)
	}

	//First check OS and Architecture
	binary := plugin.GetBinary(strings.ToLower(sdk.GOOS), strings.ToLower(sdk.GOARCH))
	if binary == nil {
		return sdk.Result{}, fmt.Errorf("unable to retrieve the plugin for release on integration %s... Aborting", pfName.Value)
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
