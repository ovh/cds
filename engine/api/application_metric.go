package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getApplicationMetricHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		metricName := vars["metricName"]

		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName)
		if errA != nil {
			return sdk.WrapError(errA, "getApplicationMetricHandler> unable to load application")
		}

		result, err := metrics.GetMetrics(api.mustDB(), key, app.ID, metricName)
		if err != nil {
			return sdk.WrapError(err, "Cannot get metrics")

		}
		return service.WriteJSON(w, result, http.StatusOK)
	}
}
