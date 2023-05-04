package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getPluginHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.pluginRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pluginName := vars["name"]

			pl, err := plugin.LoadByName(ctx, api.mustDB(), pluginName)
			if err != nil {
				return err
			}
			return service.WriteMarshal(w, req, pl, http.StatusOK)
		}
}

func (api *API) postImportPluginHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalPluginManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			var p sdk.GRPCPlugin
			db := api.mustDB()
			if err := service.UnmarshalBody(r, &p); err != nil {
				return sdk.WithStack(err)
			}
			if err := p.Validate(); err != nil {
				return sdk.WithStack(err)
			}
			p.Binaries = nil

			tx, err := db.Begin()
			if err != nil {
				return sdk.WrapError(err, "cannot start transaction")
			}
			defer tx.Rollback() //nolint

			if err := plugin.Insert(tx, &p); err != nil {
				return sdk.WrapError(err, "unable to insert plugin")
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteJSON(w, p, http.StatusOK)
		}
}