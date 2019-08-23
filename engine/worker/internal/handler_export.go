package internal

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func addBuildVarHandler(wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get body
		data, errra := ioutil.ReadAll(r.Body)
		if errra != nil {
			log.Error("addBuildVarHandler> Cannot ReadAll err: %s", errra)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var v sdk.Variable
		if err := json.Unmarshal(data, &v); err != nil {
			log.Error("addBuildVarHandler> Cannot Unmarshal err: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		v.Name = "cds.build." + v.Name

		wk.currentJob.newVariables = append(wk.currentJob.newVariables, v)
		log.Debug("Variable %s added to %+v", v.Name, wk.currentJob.newVariables)
	}
}
