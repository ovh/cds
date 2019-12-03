package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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

		var art sdk.WorkflowNodeRunArtifact
		if err := json.Unmarshal(data, &art); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		a := sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "path",
					Type:  sdk.StringParameter,
					Value: art.Name,
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
		result, err := action.RunArtifactUpload(ctx, wk, a, wk.currentJob.wJob.Parameters, wk.currentJob.secrets)
		if err != nil {
			wk.SendLog(ctx, workerruntime.LevelError, fmt.Sprintf("Artifact upload failed: %v", err))
			log.Error(ctx, "Artifact upload failed: %v", err)
			writeError(w, r, err)
			return
		}
		if result.Status != sdk.StatusSuccess {
			wk.SendLog(ctx, workerruntime.LevelError, fmt.Sprintf("Artifact upload failed: %s", result.Reason))
			log.Error(ctx, "Artifact upload failed: %v", result)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
