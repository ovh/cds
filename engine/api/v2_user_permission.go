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
			entries, err := rbac.LoadEntriesByUserID(ctx, api.mustDB(), u.ID)
			if err != nil {
				return sdk.WrapError(err, "cannot load rbac for user %s", username)
			}
			if err := NewRBACLoader(api.mustDB()).FillRBACWithNames(ctx, &entries); err != nil {
				return err
			}
			return service.WriteJSON(w, sdk.RBACsToPermissionSummary([]sdk.RBAC{entries}), http.StatusOK)
		}
}
