package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/service"
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
