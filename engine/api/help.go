package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
)

// getHelpHandler returns help informations
func (api *API) getHelpHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, api.Config.Help, http.StatusOK)
	}
}
