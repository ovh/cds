package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

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
		if err := json.Unmarshal(data, &reqArgs); err != nil {
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
		if err := wk.Client().QueueWorkflowRunResultCheck(ctx, runJobID, runResultCheck); err != nil {
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
