package workerruntime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func V2_outputHandler(ctx context.Context, wk Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		btes, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
			return
		}

		var output OutputRequest
		if err := sdk.JSONUnmarshal(btes, &output); err != nil {
			writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
			return
		}

		// Create step output
		wk.AddStepOutput(ctx, output.Name, output.Value)

		// Create run result
		if !output.StepOnly {
			result := V2RunResultRequest{
				RunResult: &sdk.V2WorkflowRunResult{
					IssuedAt:         time.Now(),
					Status:           sdk.StatusSuccess,
					WorkflowRunID:    output.WorkflowRunID,
					WorkflowRunJobID: output.WorkflowRunJobID,
					Type:             sdk.V2WorkflowRunResultTypeVariable,
					Detail: sdk.V2WorkflowRunResultDetail{
						Data: sdk.V2WorkflowRunResultVariableDetail{
							Name:  output.Name,
							Value: output.Value,
						},
					},
				},
			}

			response, err := wk.V2AddRunResult(ctx, result)
			if err != nil {
				writeError(w, r, err)
				return
			}
			log.Info(ctx, "run result %s created", response.RunResult.ID)
		}

		writeJSON(w, nil, http.StatusNoContent)
	}
}

func V2_jobRunHandler(ctx context.Context, wk Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, wk.V2GetJobRun(r.Context()), http.StatusOK)
		default:
			writeError(w, r, sdk.ErrMethodNotAllowed)
			return
		}
	}
}

func V2_projectKeyHandler(ctx context.Context, wk Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		switch r.Method {
		case http.MethodGet:
			k, err := wk.V2GetProjectKey(r.Context(), name, true)
			if err != nil {
				writeError(w, r, err)
				return
			}
			writeJSON(w, k, http.StatusOK)
		default:

			return
		}
	}
}

func V2_contextHandler(ctx context.Context, wk Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, wk.V2GetJobContext(r.Context()), http.StatusOK)
		default:
			writeError(w, r, sdk.ErrMethodNotAllowed)
			return
		}
	}
}

func V2_runResultHandler(ctx context.Context, wk Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		btes, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
			return
		}
		defer r.Body.Close()

		switch r.Method {
		case http.MethodGet:
			var filter V2FilterRunResult
			if err := sdk.JSONUnmarshal(btes, &filter); err != nil {
				writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
				return
			}
			response, err := wk.V2GetRunResult(ctx, filter)
			if err != nil {
				writeError(w, r, err)
				return
			}
			writeJSON(w, response, http.StatusOK)
		case http.MethodPost:
			var runResultRequest V2RunResultRequest
			if err := sdk.JSONUnmarshal(btes, &runResultRequest); err != nil {
				writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
				return
			}
			response, err := wk.V2AddRunResult(ctx, runResultRequest)
			if err != nil {
				writeError(w, r, err)
				return
			}
			log.Info(ctx, "run result %s created", response.RunResult.ID)
			writeJSON(w, response, http.StatusCreated)
		case http.MethodPut:
			var runResultRequest V2RunResultRequest
			if err := sdk.JSONUnmarshal(btes, &runResultRequest); err != nil {
				writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
				return
			}
			response, err := wk.V2UpdateRunResult(ctx, runResultRequest)
			if err != nil {
				writeError(w, r, err)
				return
			}
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

func writeError(w http.ResponseWriter, _ *http.Request, err error) {
	writePlainText(w, err.Error(), 500)
}

func writePlainText(w http.ResponseWriter, data string, status int) {
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(data))
}
