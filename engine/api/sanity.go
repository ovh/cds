package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getUserWarningsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		al := r.Header.Get("Accept-Language")

		var warnings []sdk.Warning
		var err error
		if getUser(ctx).Admin {
			warnings, err = sanity.LoadAllWarnings(api.MustDB(), al)
		} else {
			warnings, err = sanity.LoadUserWarnings(api.MustDB(), al, getUser(ctx).ID)
		}
		if err != nil {
			log.Warning("getUserWarnings> Cannot load user %d warnings: %s\n", getUser(ctx).ID, err)
			return err

		}

		return WriteJSON(w, r, warnings, http.StatusOK)
	}
}
