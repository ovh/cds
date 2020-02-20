package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getUserGroupsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsernamePublic"]

		u, err := user.LoadByUsername(ctx, api.mustDB(), username)
		if err != nil {
			return sdk.WrapError(err, "cannot load user %s", username)
		}

		// Get all links group user for user id
		links, err := group.LoadLinksGroupUserForUserIDs(ctx, api.mustDB(), []string{u.ID})
		if err != nil {
			return err
		}
		mLinks := make(map[int64]group.LinkGroupUser, len(links))
		for i := range links {
			mLinks[links[i].GroupID] = links[i]
		}

		// Load all groups for links and add role data
		groups, err := group.LoadAllByIDs(ctx, api.mustDB(), links.ToGroupIDs())
		if err != nil {
			return err
		}
		for i := range groups {
			groups[i].Admin = mLinks[groups[i].ID].Admin
		}

		return service.WriteJSON(w, groups, http.StatusOK)
	}
}
