package workerruntime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func V2_cacheLinkHandler(ctx context.Context, wk Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		cacheKey, err := url.PathUnescape(vars["cacheKey"])
		if err != nil {
			writeError(w, r, sdk.ErrMethodNotAllowed)
			return
		}
		switch r.Method {
		case http.MethodGet:
			cdnLinks, err := wk.V2GetCacheLink(r.Context(), cacheKey)
			if err != nil && !strings.Contains(err.Error(), "resource not found") {
				writeError(w, r, err)
				return
			} else if err != nil && strings.Contains(err.Error(), "resource not found") {
				cdnLinks = &sdk.CDNItemLinks{}
			}
			cdnLinks.CDNHttpURL = wk.CDNHttpURL()
			writeJSON(w, cdnLinks, http.StatusOK)
		default:
			writeError(w, r, sdk.ErrMethodNotAllowed)
			return
		}
	}

}

func V2_cacheSignatureHandler(ctx context.Context, wk Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		cacheKey, err := url.PathUnescape(vars["cacheKey"])
		if err != nil {
			writeError(w, r, sdk.ErrMethodNotAllowed)
			return
		}
		switch r.Method {
		case http.MethodGet:
			sign, err := wk.V2GetCacheSignature(r.Context(), cacheKey)
			if err != nil {
				writeError(w, r, err)
				return
			}
			writeJSON(w, sign, http.StatusOK)
		default:
			writeError(w, r, sdk.ErrMethodNotAllowed)
			return
		}
	}

}

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

func V2_runResultsSynchronizeHandler(ctx context.Context, wk Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if err := wk.V2RunResultsSynchronize(ctx); err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, "run results synchronized", http.StatusOK)
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
