package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// ConfigUserHandler return url of CDS UI
func (api *API) configUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.ConfigUser{
			URLUI:  api.Config.URL.UI,
			URLAPI: api.Config.URL.API,
		}, http.StatusOK)
	}
}

func (api *API) configCDNHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tcpURL, tcpURLEnableTLS, err := services.GetCDNPublicTCPAdress(ctx, api.mustDB())
		if err != nil {
			return err
		}
		httpURL, err := services.GetCDNPublicHTTPAdress(ctx, api.mustDB())
		if err != nil {
			return err
		}
		return service.WriteJSON(w,
			sdk.CDNConfig{TCPURL: tcpURL,
				TCPURLEnableTLS: tcpURLEnableTLS,
				HTTPURL:         httpURL},
			http.StatusOK)
	}
}

func (api *API) configAPIHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.APIConfig{
			DefaultRunRetentionPolicy:    api.Config.Workflow.DefaultRetentionPolicy,
			ProjectCreationDisabled:      api.Config.Project.CreationDisabled,
			ProjectInfoCreationDisabled:  api.Config.Project.InfoCreationDisabled,
			ProjectVCSManagementDisabled: api.Config.Project.VCSManagementDisabled,
		}, http.StatusOK)
	}
}
