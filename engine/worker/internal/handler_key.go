package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func keyInstallHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		keyName := vars["key"]

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
			return
		}

		defer r.Body.Close() // nolint

		var mapBody = make(map[string]string)
		if len(body) > 0 {
			if err := json.Unmarshal(body, &mapBody); err != nil {
				writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, err))
				return
			}
		}

		var key *sdk.Variable

		if wk.currentJob.secrets == nil {
			log.Error(ctx, "%v", err)
			writeError(w, r, sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Cannot find any keys for your job")))
			return
		}

		for _, k := range wk.currentJob.secrets {
			if k.Name == ("cds.key." + keyName + ".priv") {
				key = &k
				break
			}
		}

		if key == nil {
			log.Error(ctx, "%v", err)
			writeError(w, r, sdk.NewError(sdk.ErrNotFound, fmt.Errorf("Cannot find any keys for your job")))
			return
		}

		filename := mapBody["file"]
		response, err := keyInstall(wk, filename, key)
		if err != nil {
			log.Error(ctx, "Unable to install key %s: %v", key.Name, err)
			if sdkerr, ok := err.(*sdk.Error); ok {
				writeError(w, r, sdkerr)
			} else {
				writeError(w, r, sdk.NewError(sdk.ErrUnknownError, err))
			}
			return
		}
		log.Debug("key %s installed to %s", key.Name, response.PKey)
		writeJSON(w, response, 200)
	}
}

func keyInstall(wk workerruntime.Runtime, filename string, key *sdk.Variable) (*workerruntime.KeyResponse, error) {
	if filename == "" {
		return wk.InstallKey(*key)
	}

	log.Debug("worker.keyInstall> installing key %s to %s", key.Name, filename)

	if !sdk.PathIsAbs(filename) {
		return nil, fmt.Errorf("unsupported relative path")
	}

	log.Debug("worker.keyInstall> destination: %s", filename)
	return wk.InstallKeyTo(*key, filename)
}
