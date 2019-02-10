package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

func (wk *currentWorker) serviceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceType := vars["type"]

	uri := "/services/" + serviceType

	log.Info("serviceHandler> Getting service configuration...")

	resp, _, code, err := wk.client.(cdsclient.Raw).Request(r.Context(), "GET", uri, nil)
	if err == nil && code < 300 {
		writeByteArray(w, resp, http.StatusOK)
		return
	}

	writeError(w, r, fmt.Errorf("serviceHandler> Unable to get service configuration: %v", err))
}
