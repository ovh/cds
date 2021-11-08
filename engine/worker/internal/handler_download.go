package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/internal/action"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func downloadHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		// Get body
		data, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, errRead)
			writeError(w, r, newError)
			return
		}
		defer r.Body.Close() // nolint

		var reqArgs workerruntime.DownloadArtifact
		if err := sdk.JSONUnmarshal(data, &reqArgs); err != nil {
			newError := sdk.NewError(sdk.ErrWrongRequest, err)
			writeError(w, r, newError)
			return
		}

		a := sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "path",
					Type:  sdk.StringParameter,
					Value: reqArgs.Destination,
				},
				{
					Name:  "pattern",
					Type:  sdk.StringParameter,
					Value: reqArgs.Pattern,
				},
				{
					Name:  "tag",
					Type:  sdk.StringParameter,
					Value: reqArgs.Tag,
				},
				{
					Name:  "workflow",
					Type:  sdk.StringParameter,
					Value: reqArgs.Workflow,
				},
				{
					Name:  "number",
					Type:  sdk.NumberParameter,
					Value: strconv.Itoa(int(reqArgs.Number)),
				},
			},
		}

		workingDir, err := workerruntime.WorkingDirectory(wk.currentJob.context)
		if err != nil {
			log.Error(ctx, "Artifact upload failed: No working directory: %v", err)
			writeError(w, r, err)
			return
		}
		ctx = workerruntime.SetWorkingDirectory(ctx, workingDir)

		result, err := action.RunArtifactDownload(ctx, wk, a, wk.currentJob.secrets)
		if err != nil {
			log.Error(ctx, "unable to upload artifacts: %v", err)
			writeError(w, r, err)
			return
		}
		if result.Status != sdk.StatusSuccess {
			log.Error(ctx, "Artifact upload failed: %v", result)
			writeError(w, r, fmt.Errorf("artifact upload failed: %s", result.Reason))
			return
		}
	}
}
