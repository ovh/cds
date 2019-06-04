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

		if JWT(ctx).AuthentifiedUser.Username != username && !isAdmin(ctx) {
			return service.WriteJSON(w, nil, http.StatusForbidden)
		}

		usr, err := user.LoadByUsername(ctx, api.mustDB(), username, user.LoadOptions.WithDeprecatedUser)
		if err != nil {
			return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Cannot load user %s", username)
		}

		var groups, groupsAdmin []sdk.Group

		var err1, err2 error
		groups, err1 = group.LoadGroupByUser(api.mustDB(), usr.OldUserStruct.ID)
		if err1 != nil {
			return sdk.WrapError(err1, "getUserGroupsHandler: Cannot load group by user")
		}

		groupsAdmin, err2 = group.LoadGroupByAdmin(api.mustDB(), usr.OldUserStruct.ID)
		if err2 != nil {
			return sdk.WrapError(err2, "getUserGroupsHandler: Cannot load group by admin")
		}

		res := map[string][]sdk.Group{}
		res["groups"] = groups
		res["groups_admin"] = groupsAdmin

		return service.WriteJSON(w, res, http.StatusOK)
	}
}
