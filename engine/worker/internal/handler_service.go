package internal

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"
)

func serviceHandler(ctx context.Context, wk *CurrentWorker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		serviceType := vars["type"]

		log.Debug(ctx, "Getting service configuration...")
		servicesConfig, err := wk.Client().ServiceConfigurationGet(ctx, serviceType)
		if err != nil {
			log.Warn(ctx, "unable to get data: %v", err)
			writeError(w, r, fmt.Errorf("unable to get service configuration"))
		}
		writeJSON(w, servicesConfig, http.StatusOK)
		return
	}
}
