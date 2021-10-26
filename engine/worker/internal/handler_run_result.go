package internal

import (
	"context"
	"fmt"
	"io"
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

func addRunResultArtifactManagerHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addRunResult(ctx, wk, w, r, sdk.WorkflowRunResultTypeArtifactManager)
	}
}

func addRunResultStaticFileHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		addRunResult(ctx, wk, w, r, sdk.WorkflowRunResultTypeStaticFile)
	}
}

func addRunResult(ctx context.Context, wk *CurrentWorker, w http.ResponseWriter, r *http.Request, stype sdk.WorkflowRunResultType) {
	ctx = workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
	ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
	ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

	data, err := io.ReadAll(r.Body)
	if err != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, err)
		writeError(w, r, newError)
		return
	}
	defer r.Body.Close() //nolint

	var name string
	switch stype {
	case sdk.WorkflowRunResultTypeStaticFile:
		var reqArgs sdk.WorkflowRunResultStaticFile
		if err := sdk.JSONUnmarshal(data, &reqArgs); err != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, err)
			writeError(w, r, newError)
			return
		}
		name = reqArgs.Name
	case sdk.WorkflowRunResultTypeArtifactManager:
		var reqArgs sdk.WorkflowRunResultArtifactManager
		if err := sdk.JSONUnmarshal(data, &reqArgs); err != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, err)
			writeError(w, r, newError)
			return
		}
		name = reqArgs.Name
	}

	runID, runNodeID, runJobID := wk.GetJobIdentifiers()
	runResultCheck := sdk.WorkflowRunResultCheck{
		RunJobID:   runJobID,
		RunNodeID:  runNodeID,
		RunID:      runID,
		Name:       name,
		ResultType: stype,
	}
	code, err := wk.Client().QueueWorkflowRunResultCheck(ctx, runJobID, runResultCheck)
	if err != nil {
		if code == 409 {
			writeError(w, r, sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to upload the same file twice: %s", name))
			return
		}
		writeError(w, r, sdk.WrapError(err, "unable to check run result %s", name))
		return
	}

	addRunRequest := sdk.WorkflowRunResult{
		Type:              stype,
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
