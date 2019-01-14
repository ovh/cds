package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// ConfigUserHandler return url of CDS UI
func (api *API) ConfigUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, map[string]string{sdk.ConfigURLAPIKey: api.Config.URL.API, sdk.ConfigURLUIKey: api.Config.URL.UI}, http.StatusOK)
	}
}

// ConfigVCShandler return the configuration of vcs server
func (api *API) ConfigVCShandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vcsServers, err := repositoriesmanager.LoadAll(ctx, api.mustDB(), api.Cache)
		if err != nil {
			return sdk.WrapError(err, "error")
		}
		return service.WriteJSON(w, vcsServers, http.StatusOK)
	}
}
