package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"

	"github.com/ovh/cds/sdk"
)

func artifactsHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		data, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, errRead)
			writeError(w, r, newError)
			return
		}
		defer r.Body.Close()

		var reqArgs workerruntime.DownloadArtifact
		if err := sdk.JSONUnmarshal(data, &reqArgs); err != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, err)
			writeError(w, r, newError)
			return
		}

		if reqArgs.Workflow == "" {
			reqArgs.Workflow = sdk.ParameterValue(wk.currentJob.params, "cds.workflow")
		}

		if reqArgs.Number == 0 {
			var errN error
			buildNumberString := sdk.ParameterValue(wk.currentJob.params, "cds.run.number")
			reqArgs.Number, errN = strconv.ParseInt(buildNumberString, 10, 64)
			if errN != nil {
				newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("cannot parse '%s' as run number: %s", buildNumberString, errN))
				writeError(w, r, newError)
				return
			}
		}

		projectKey := sdk.ParameterValue(wk.currentJob.params, "cds.project")

		regexp, errp := regexp.Compile(reqArgs.Pattern)
		if errp != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("invalid pattern %s : %s", reqArgs.Pattern, errp))
			writeError(w, r, newError)
			return
		}

		artifactsJSON := []sdk.WorkflowNodeRunArtifact{}

		workflowRunResults, err := wk.client.WorkflowRunResultsList(ctx, projectKey, reqArgs.Workflow, reqArgs.Number)
		if err != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("cannot list workflow run result: %s", err))
			writeError(w, r, newError)
			return
		}
		for _, result := range workflowRunResults {
			if result.Type != sdk.WorkflowRunResultTypeArtifact {
				continue
			}
			artData, err := result.GetArtifact()
			if err != nil {
				newError := sdk.NewError(sdk.ErrUnknownError, fmt.Errorf("item is not an artifact: %s", err))
				writeError(w, r, newError)
				return
			}
			if reqArgs.Pattern != "" && !regexp.MatchString(artData.Name) {
				continue
			}
			artifactsJSON = append(artifactsJSON, sdk.WorkflowNodeRunArtifact{
				MD5sum:  artData.MD5,
				Name:    artData.Name,
				Size:    artData.Size,
				Created: result.Created,
				Perm:    artData.Perm,
			})
		}

		writeJSON(w, artifactsJSON, http.StatusOK)
	}
}
