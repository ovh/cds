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

		app, err := application.LoadByName(ctx, api.mustDB(), key, appName)
		if err != nil {
			return err
		}

		result, err := metrics.GetMetrics(ctx, api.mustDB(), key, app.ID, metricName)
		if err != nil {
			return sdk.WrapError(err, "cannot get metrics")

		}
		return service.WriteJSON(w, result, http.StatusOK)
	}
}
