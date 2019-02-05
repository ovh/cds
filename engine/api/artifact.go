package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getArtifactsStoreHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		projectKey := vars["projectKey"]
		integrationName := vars["integrationName"]

		if integrationName != sdk.DefaultStorageIntegrationName {
			projectIntegration, err := integration.LoadProjectIntegrationByName(api.mustDB(), projectKey, integrationName, false)
			if err != nil {
				return sdk.WrapError(err, "Cannot load projectIntegration %s/%s", projectKey, integrationName)
			}
			// TODO YESNAULT
			s := sdk.ArtifactsStore{
				Name:                  projectIntegration.Name,
				TemporaryURLSupported: api.SharedStorage.TemporaryURLSupported(),
			}
			return service.WriteJSON(w, s, http.StatusOK)
		}

		s := sdk.ArtifactsStore{
			TemporaryURLSupported: api.SharedStorage.TemporaryURLSupported(),
		}
		return service.WriteJSON(w, s, http.StatusOK)
	}
}
