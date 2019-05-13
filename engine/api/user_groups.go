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

// getUserGroupsHandler returns groups of the user
func (api *API) getUserGroupsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]

		if !deprecatedGetUser(ctx).Admin && username != deprecatedGetUser(ctx).Username {
			return service.WriteJSON(w, nil, http.StatusForbidden)
		}

		usr, err := user.LoadUserByUsername(api.mustDB(), username)
		if err != nil {
			return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Cannot load user %s", username)
		}

		u, err := user.GetDeprecatedUser(api.mustDB(), usr)
		if err != nil {
			return err
		}

		var groups, groupsAdmin []sdk.Group

		var err1, err2 error
		groups, err1 = group.LoadGroupByUser(api.mustDB(), u.ID)
		if err1 != nil {
			return sdk.WrapError(err1, "getUserGroupsHandler: Cannot load group by user")
		}

		groupsAdmin, err2 = group.LoadGroupByAdmin(api.mustDB(), u.ID)
		if err2 != nil {
			return sdk.WrapError(err2, "getUserGroupsHandler: Cannot load group by admin")
		}

		res := map[string][]sdk.Group{}
		res["groups"] = groups
		res["groups_admin"] = groupsAdmin

		return service.WriteJSON(w, res, http.StatusOK)
	}
}
