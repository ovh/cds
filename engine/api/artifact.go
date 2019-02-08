package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getStorageDriver(projectKey, integrationName string) (objectstore.Driver, error) {
	var storageDriver objectstore.Driver
	if integrationName != sdk.DefaultStorageIntegrationName {
		var err error
		storageDriver, err = objectstore.InitDriver(api.mustDB(), projectKey, integrationName)
		if err != nil {
			return nil, sdk.WrapError(err, "Cannot load storage driver %s/%s", projectKey, integrationName)
		}
	} else {
		storageDriver = api.SharedStorage
	}
	return storageDriver, nil
}

func (api *API) getArtifactsStoreHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		projectKey := vars["permProjectKey"]
		integrationName := vars["integrationName"]

		if integrationName != sdk.DefaultStorageIntegrationName {
			storageDriver, err := objectstore.InitDriver(api.mustDB(), projectKey, integrationName)
			if err != nil {
				return sdk.WrapError(err, "Cannot init storage driver")
			}
			s := sdk.ArtifactsStore{
				Name:                  storageDriver.GetProjectIntegration().Name,
				TemporaryURLSupported: storageDriver.TemporaryURLSupported(),
			}
			return service.WriteJSON(w, s, http.StatusOK)
		}

		s := sdk.ArtifactsStore{
			Name:                  api.SharedStorage.GetProjectIntegration().Name,
			TemporaryURLSupported: api.SharedStorage.TemporaryURLSupported(),
		}
		return service.WriteJSON(w, s, http.StatusOK)
	}
}
