package api

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/link"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"net/http"
)

func (api *API) getUserLinksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		username := vars["permUsername"]

		u, err := user.LoadByUsername(ctx, api.mustDB(), username)
		if err != nil {
			return err
		}

		links, err := link.LoadUserLinksByUserID(ctx, api.mustDB(), u.ID)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, links, http.StatusOK)
	}
}
