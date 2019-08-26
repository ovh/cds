package internal

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var keysDirectory string

func keyInstallHandler(wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		keyName := vars["key"]
		fileName := r.FormValue("file")
		var key *sdk.Variable

		if wk.currentJob.secrets == nil {
			err := sdk.Error{
				Message: "Cannot find any keys for your job",
				Status:  http.StatusBadRequest,
			}
			log.Error("%v", err)
			writeJSON(w, err, err.Status)
			return
		}

		for _, k := range wk.currentJob.secrets {
			if k.Name == ("cds.key." + keyName + ".priv") {
				key = &k
				break
			}
		}

		if key == nil {
			err := sdk.Error{
				Message: fmt.Sprintf("Key %s not found", keyName),
				Status:  http.StatusNotFound,
			}
			log.Error("%v", err)
			writeJSON(w, err, err.Status)
			return
		}

		response, err := wk.InstallKey(*key, fileName)
		if err != nil {
			log.Error("Unable to install key %s: %v", key.Name, err)
			if err, ok := err.(*sdk.Error); ok {
				writeJSON(w, err, err.Status)
			} else {
				err := sdk.Error{
					Message: err.Error(),
					Status:  sdk.ErrUnknownError.Status,
				}
				writeJSON(w, err, err.Status)
			}
			return
		}
		writeJSON(w, response, 200)
	}
}
