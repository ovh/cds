package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// DEPRECATED
// TODO: remove this code after CDN would be mandatory
func (api *API) getArtifactsStoreHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		integrationName := vars["integrationName"]

		storageDriver, err := objectstore.GetDriver(ctx, api.mustDB(), api.SharedStorage, projectKey, integrationName)
		if err != nil {
			return sdk.WrapError(err, "Cannot init storage driver")
		}
		s := sdk.ArtifactsStore{
			Name:                  storageDriver.GetProjectIntegration().Name,
			TemporaryURLSupported: storageDriver.TemporaryURLSupported(),
		}
		return service.WriteJSON(w, s, http.StatusOK)
	}
}
