package internal

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func getRunResultHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		projectKey := sdk.ParameterValue(wk.currentJob.params, "cds.project")
		wName := sdk.ParameterValue(wk.currentJob.params, "cds.workflow")
		runNumberString := sdk.ParameterValue(wk.currentJob.params, "cds.run.number")
		runNumber, err := strconv.ParseInt(runNumberString, 10, 64)
		if err != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("cannot parse '%s' as run number: %s", runNumberString, err))
			writeError(w, r, newError)
			return
		}
		results, err := wk.Client().WorkflowRunResultsList(ctx, projectKey, wName, runNumber)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, results, http.StatusOK)
	}
}

func addRunResulthandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, errRead)
			writeError(w, r, newError)
			return
		}
		defer r.Body.Close() //nolint

		var reqArgs sdk.WorkflowRunResultArtifactManager
		if err := sdk.JSONUnmarshal(data, &reqArgs); err != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, err)
			writeError(w, r, newError)
			return
		}
		runID, runNodeID, runJobID := wk.GetJobIdentifiers()
		runResultCheck := sdk.WorkflowRunResultCheck{
			RunJobID:   runJobID,
			RunNodeID:  runNodeID,
			RunID:      runID,
			Name:       reqArgs.Name,
			ResultType: sdk.WorkflowRunResultTypeArtifactManager,
		}
		code, err := wk.Client().QueueWorkflowRunResultCheck(ctx, runJobID, runResultCheck)
		if err != nil {
			if code == 409 {
				writeError(w, r, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to upload the same file twice"))
				return
			}
			writeError(w, r, sdk.WrapError(err, "unable to check run result %s", reqArgs.Name))
			return

		}

		addRunRequest := sdk.WorkflowRunResult{
			Type:              sdk.WorkflowRunResultTypeArtifactManager,
			DataRaw:           data,
			Created:           time.Now(),
			WorkflowRunJobID:  runJobID,
			WorkflowRunID:     runID,
			WorkflowNodeRunID: runNodeID,
		}
		if err := wk.client.QueueWorkflowRunResultsAdd(ctx, wk.currentJob.wJob.ID, addRunRequest); err != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("cannot add run result: %s", err))
			writeError(w, r, newError)
			return
		}
		writeJSON(w, nil, http.StatusOK)
	}
}
