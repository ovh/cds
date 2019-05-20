package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/bookmark"
	"github.com/ovh/cds/engine/service"
)

func (api *API) getBookmarksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		u := getAPIConsumer(ctx)
		if !u.IsRealUser() {
			return sdk.WithStack(sdk.ErrForbidden)
		}
		data, err := bookmark.LoadAll(api.mustDB(), u.OnBehalfOf.OldUserStruct)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, data, http.StatusOK)
	}
}
