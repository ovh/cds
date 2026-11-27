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
			entries, err := rbac.LoadEntriesByGroupID(ctx, api.mustDB(), gp.ID)
			if err != nil {
				return sdk.WrapError(err, "cannot load rbac for group %s", groupName)
			}
			if err := NewRBACLoader(api.mustDB()).FillRBACWithNames(ctx, &entries); err != nil {
				return err
			}
			return service.WriteJSON(w, sdk.RBACsToPermissionSummary([]sdk.RBAC{entries}), http.StatusOK)
		}
}
