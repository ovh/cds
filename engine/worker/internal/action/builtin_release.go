package action

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/internal/plugin"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
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

	pluginClient, err := plugin.NewClient(ctx, wk, plugin.TypeIntegration, sdk.GRPCPluginRelease, plugin.InputManagementDefault)
	if err != nil {
		return sdk.Result{Status: sdk.StatusFail, Reason: fmt.Sprintf("unable to create plugin: %v", err)}, nil
	}
	defer pluginClient.Close(ctx)

	opts := sdk.ParametersToMap(wk.Parameters())

	for _, v := range a.Parameters {
		opts[v.Name] = v.Value
	}
	pluginResult := pluginClient.Run(ctx, opts)

	return sdk.Result{Status: pluginResult.Status, Reason: pluginResult.Details}, nil
}
