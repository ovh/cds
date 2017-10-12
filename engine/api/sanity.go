package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/sdk"
)

func (api *API) getUserWarningsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		al := r.Header.Get("Accept-Language")

		var warnings []sdk.Warning
		var err error
		if getUser(ctx).Admin {
			warnings, err = sanity.LoadAllWarnings(api.mustDB(), al)
		} else {
			warnings, err = sanity.LoadUserWarnings(api.mustDB(), al, getUser(ctx).ID)
		}
		if err != nil {
			return sdk.WrapError(err, "getUserWarnings> Cannot load user %d warnings", getUser(ctx).ID)

		}

		return WriteJSON(w, r, warnings, http.StatusOK)
	}
}
