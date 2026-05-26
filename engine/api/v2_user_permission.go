package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// getUserPermissionHandler Get user permissions
func (api *API) getUserPermissionHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isCurrentUser),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			username := vars["user"]
			u, err := user.LoadByUsername(ctx, api.mustDB(), username)
			if err != nil {
				return sdk.WrapError(err, "cannot load user %s", username)
			}
			permissions, err := rbac.LoadAllRBACByUserID(ctx, api.mustDB(), u.ID, rbac.LoadOptions.All)
			if err != nil {
				return sdk.WrapError(err, "cannot load rbac for user %s", username)
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
