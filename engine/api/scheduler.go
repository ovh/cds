package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
)

func (api *API) getSchedulerApplicationPipelineHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, nil, http.StatusOK)
	}
}
