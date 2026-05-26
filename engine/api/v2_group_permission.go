package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// getGroupPermissionHandler Get group permissions
func (api *API) getGroupPermissionHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBACNone(),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			groupName := vars["name"]

			gp, err := group.LoadByName(ctx, api.mustDB(), groupName)
			if err != nil {
				return sdk.WrapError(err, "cannot load group %s", groupName)
			}
			permissions, err := rbac.LoadAllRBACByGroupID(ctx, api.mustDB(), gp.ID, rbac.LoadOptions.All)
			if err != nil {
				return sdk.WrapError(err, "cannot load rbac for user %s", groupName)
			}

			rbacLoader := NewRBACLoader(api.mustDB())
			for i := range permissions {
				perm := &permissions[i]
				if err := rbacLoader.FillRBACWithNames(ctx, perm); err != nil {
					return err
				}
			}
			return service.WriteJSON(w, sdk.RBACsToPermissionSummary(permissions), http.StatusOK)
		}
}
