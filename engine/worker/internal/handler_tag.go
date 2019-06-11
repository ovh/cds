package internal

import (
	"context"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk"
)

func tagHandler(wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm() // Parses the request body
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
