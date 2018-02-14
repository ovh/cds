package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/ui"
	"github.com/ovh/cds/sdk"
)

func (api *API) getUINavbarHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		data, err := ui.LoadNavbarData(api.mustDB(), api.Cache, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getUINavbarHandler")
		}
		return WriteJSON(w, data, http.StatusOK)
	}
}
