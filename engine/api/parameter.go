package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (api *API) getVariableTypeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, sdk.AvailableVariableType, http.StatusOK)
	}
}

func (api *API) getParameterTypeHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, sdk.AvailableParameterType, http.StatusOK)
	}
}
