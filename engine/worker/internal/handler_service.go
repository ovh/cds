package internal

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func serviceHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		serviceType := vars["type"]

		var lasterr error
		var code int
		var serviceConfig *sdk.ExternalServiceConfiguration
		for try := 1; try <= 10; try++ {
			log.Debug("Getting service configuration...")
			serviceConfig, lasterr = wk.Client().ServiceConfigurationGet(ctx, serviceType)
			if lasterr == nil && code < 300 {
				writeJSON(w, serviceConfig, http.StatusOK)
				return
			}
			log.Warning(ctx, "cannot get external service configuration: HTTP %d err: %s - try: %d - new try in 5s", code, lasterr, try)
			time.Sleep(5 * time.Second)
		}
		writeError(w, r, fmt.Errorf("unable to get service configuration"))
	}
}
