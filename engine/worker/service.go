package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (wk *currentWorker) serviceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceType := vars["type"]

	uri := "/services/" + serviceType

	var lasterr error
	var code int
	var resp []byte
	for try := 1; try <= 10; try++ {
		log.Info("serviceHandler> Getting service configuration...")
		resp, code, lasterr = sdk.Request("GET", uri, nil)
		if lasterr == nil && code < 300 {
			writeJSON(w, resp, http.StatusOK)
			return
		}
		log.Warning("serviceHandler> Cannot get serviceconfiguration: HTTP %d err: %s - try: %d - new try in 5s", code, lasterr, try)
		time.Sleep(5 * time.Second)
	}
	writeError(w, r, fmt.Errorf("serviceHandler> Unable to get service configuration"))
}
