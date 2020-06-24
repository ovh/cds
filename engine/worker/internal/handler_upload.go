package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/ovh/cds/engine/worker/internal/action"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func uploadHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var art workerruntime.UploadArtifact
		if err := json.Unmarshal(data, &art); err != nil {
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

		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		workingDir, err := workerruntime.WorkingDirectory(wk.currentJob.context)
		if err != nil {
			wk.SendLog(ctx, workerruntime.LevelError, fmt.Sprintf("Artifact upload failed: %v", err))
			log.Error(ctx, "Artifact upload failed: No working directory: %v", err)
			writeError(w, r, err)
			return
		}
		ctx = workerruntime.SetWorkingDirectory(ctx, workingDir)

		result, err := action.RunArtifactUpload(ctx, wk, a, wk.currentJob.secrets)
		if err != nil {
			wk.SendLog(ctx, workerruntime.LevelError, fmt.Sprintf("Artifact upload failed: %v", err))
			log.Error(ctx, "unable to upload artifacts: %v", err)
			writeError(w, r, err)
			return
		}
		if result.Status != sdk.StatusSuccess {
			wk.SendLog(ctx, workerruntime.LevelError, fmt.Sprintf("Artifact upload failed: %s", result.Reason))
			log.Error(ctx, "Artifact upload failed: %v", result)
			writeError(w, r, fmt.Errorf("Artifact upload failed: %s", result.Reason))
			return
		}
	}
}
