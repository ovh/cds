package internal

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func setVersionHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
			return
		}
		defer r.Body.Close()

		var req workerruntime.CDSVersionSet
		if err := json.Unmarshal(data, &req); err != nil {
			writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
			return
		}

		if req.Value == "" {
			writeError(w, r, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given CDS version value"))
			return
		}

		if err := wk.client.QueueJobSetVersion(ctx, wk.currentJob.wJob.ID, sdk.WorkflowRunVersion{
			Value: req.Value,
		}); err != nil {
			writeError(w, r, err)
			return
		}

		// Override cds.version value in params to allow usage of this value in others steps
		for i := range wk.currentJob.params {
			if wk.currentJob.params[i].Name == "cds.version" {
				wk.currentJob.params[i].Value = req.Value
				break
			}
		}
	}
}
