package workerruntime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func V2_runResultHandler(ctx context.Context, wk Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		btes, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
			return
		}

		var runResultRequest V2RunResultRequest
		if err := sdk.JSONUnmarshal(btes, &runResultRequest); err != nil {
			writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
			return
		}

		switch r.Method {
		case http.MethodPost:
			log.Info(ctx, "processing request for run result creation %+v", runResultRequest)
			response, err := wk.V2AddRunResult(ctx, runResultRequest)
			if err != nil {
				writeError(w, r, err)
				return
			}
			log.Info(ctx, "run result %s created", response.RunResult.ID)
			writeJSON(w, response, http.StatusCreated)
		case http.MethodPut:
			log.Info(ctx, "processing request for run result update %+v", runResultRequest.RunResult)
			response, err := wk.V2UpdateRunResult(ctx, runResultRequest)
			if err != nil {
				writeError(w, r, err)
				return
			}
			log.Info(ctx, "=> %+v", response)
			log.Info(ctx, "run result %s updated", response.RunResult.ID)
			writeJSON(w, response, http.StatusOK)
		default:
			writeError(w, r, sdk.ErrNotFound)
		}
	}
}

func writeJSON(w http.ResponseWriter, data interface{}, status int) {
	b, _ := json.Marshal(data)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	writePlainText(w, err.Error(), 500)
}

func writePlainText(w http.ResponseWriter, data string, status int) {
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(data))
}
