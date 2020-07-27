package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) isFeatureEanbledHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]

		if !isService(ctx) {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "not allowed")
		}

		var params map[string]string
		if err := service.UnmarshalBody(r, &params); err != nil {
			return err
		}

		enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), name, params)
		return service.WriteJSON(w, sdk.FeatureEnabledResponse{Name: name, Enabled: enabled}, http.StatusOK)
	}
}
