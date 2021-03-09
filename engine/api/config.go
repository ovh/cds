package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// ConfigUserHandler return url of CDS UI
func (api *API) ConfigUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.ConfigUser{
			URLUI:  api.Config.URL.UI,
			URLAPI: api.Config.URL.API,
		}, http.StatusOK)
	}
}

// ConfigVCShandler return the configuration of vcs server
func (api *API) ConfigVCShandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vcsServers, err := repositoriesmanager.LoadAll(ctx, api.mustDB(), api.Cache)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, vcsServers, http.StatusOK)
	}
}

func (api *API) ConfigCDNHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tcpURL, err := services.GetCDNPublicTCPAdress(ctx, api.mustDB())
		if err != nil {
			return err
		}
		httpURL, err := services.GetCDNPublicHTTPAdress(ctx, api.mustDB())
		if err != nil {
			return err
		}
		return service.WriteJSON(w, sdk.CDNConfig{
			TCPURL:  tcpURL,
			HTTPURL: httpURL,
		}, http.StatusOK)
	}
}

func (api *API) ConfigAPIHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.APIConfig{
			DefaultRunRetentionPolicy: api.Config.Workflow.DefaultRetentionPolicy,
		}, http.StatusOK)
	}
}
