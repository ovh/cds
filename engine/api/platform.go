package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/sdk"
)

func (api *API) getPlatformModels() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		p, err := platform.LoadModels(api.mustDB(ctx))
		if err != nil {
			return sdk.WrapError(err, "getPlatformModels> Cannot get platform models")
		}
		return WriteJSON(w, p, http.StatusOK)
	}
}
