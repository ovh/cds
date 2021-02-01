package internal

import (
	"context"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
)

func tagHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := workerruntime.SetJobID(ctx, wk.currentJob.wJob.ID)
		ctx = workerruntime.SetStepOrder(ctx, wk.currentJob.currentStepIndex)
		ctx = workerruntime.SetStepName(ctx, wk.currentJob.currentStepName)

		if err := r.ParseForm(); err != nil {
			writeError(w, r, err)
			return
		}
		tags := []sdk.WorkflowRunTag{}
		for k := range r.Form {
			tags = append(tags, sdk.WorkflowRunTag{
				Tag:   k,
				Value: r.Form.Get(k),
			})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := wk.client.QueueJobTag(ctx, wk.currentJob.wJob.ID, tags); err != nil {
			writeError(w, r, err)
			return
		}
	}
}
