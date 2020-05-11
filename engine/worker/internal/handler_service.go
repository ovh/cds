package internal

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"

	"github.com/ovh/cds/sdk/log"
)

func serviceHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		serviceType := vars["type"]

		log.Debug("Getting service configuration...")
		serviceConfig, err := wk.Client().ServiceConfigurationGet(ctx, serviceType)
		if err != nil {
			log.Warning(ctx, "unable to get data: %v", err)
			writeError(w, r, fmt.Errorf("unable to get service configuration"))
		}
		writeJSON(w, serviceConfig, http.StatusOK)
		return
	}
}
