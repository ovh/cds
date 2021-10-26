package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/worker/internal/action"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func uploadHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		// Get body
		data, errRead := io.ReadAll(r.Body)
		if errRead != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var art workerruntime.UploadArtifact
		if err := sdk.JSONUnmarshal(data, &art); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		artifactPath := art.Name
		if !sdk.PathIsAbs(artifactPath) {
			artifactPath = filepath.Join(art.WorkingDirectory, art.Name)
		}

		a := sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "path",
					Type:  sdk.StringParameter,
					Value: artifactPath,
				},
				{
					Name:  "tag",
					Type:  sdk.StringParameter,
					Value: art.Tag,
				},
				{
					Name:  "destination",
					Type:  sdk.StringParameter,
					Value: r.FormValue("integration"),
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

		result, err := action.RunArtifactUpload(ctx, wk, a, wk.currentJob.secrets)
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
