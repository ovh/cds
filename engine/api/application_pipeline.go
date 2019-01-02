package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// DEPRECATED
func (api *API) attachPipelineToApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return sdk.ErrMethodNotAllowed
	}
}

// DEPRECATED
func (api *API) attachPipelinesToApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return sdk.ErrMethodNotAllowed
	}
}

// DEPRECATED
func (api *API) updatePipelinesToApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return sdk.ErrMethodNotAllowed
	}
}

// DEPRECATED
func (api *API) updatePipelineToApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return sdk.ErrMethodNotAllowed
	}
}

// DEPRECATED
func (api *API) getPipelinesInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		pipelines, err := application.GetAllPipelines(api.mustDB(), key, appName)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "getPipelinesInApplicationHandler: Cannot load pipelines for application %s", appName)
		}

		return service.WriteJSON(w, pipelines, http.StatusOK)
	}
}

// DEPRECATED
func (api *API) removePipelineFromApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return sdk.ErrMethodNotAllowed

	}
}
