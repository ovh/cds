package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/platform"
)

func (api *API) getPlatformModel() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, platform.Models, http.StatusOK)
	}
}
