package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/navbar"
	"github.com/ovh/cds/sdk"
)

func (api *API) getNavbarHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		data, err := navbar.LoadNavbarData(api.mustDB(), api.Cache, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getNavbarHandler")
		}
		return WriteJSON(w, data, http.StatusOK)
	}
}
