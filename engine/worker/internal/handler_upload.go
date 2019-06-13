package internal

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ovh/cds/engine/worker/internal/action"
	"github.com/ovh/cds/sdk"
)

func uploadHandler(wk *CurrentWorker) http.HandlerFunc {
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

		result, err := action.RunArtifactUpload(context.Background(), wk, a, wk.currentJob.wJob.Parameters, wk.currentJob.secrets)
		if err != nil {
			writeError(w, r, err)
			return
		}
		if result.Status != sdk.StatusSuccess {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
