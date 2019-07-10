package api

import (
	"context"
	"net/http"

  "github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getUserContactsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsername"]

		u, err := user.LoadByUsername(ctx, api.mustDB(), username)
		if err != nil {
			return sdk.WrapError(err, "cannot load user %s", username)
		}

		contacts, err := user.LoadContactsByUserIDs(ctx, api.mustDB(), []string{u.ID})
		if err != nil {
			return err
		}

		return service.WriteJSON(w, contacts, http.StatusOK)
	}
}
