package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/vcs"
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

// ConfigVCShandler return the configuration of vcs server
func (api *API) configVCShandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vcsServers, err := repositoriesmanager.LoadAll(ctx, api.mustDB(), api.Cache)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, vcsServers, http.StatusOK)
	}
}

func (api *API) configVCSGerritHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !isHooks(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// deprecated vcs
		srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeVCS)
		if err != nil {
			return err
		}

		vcsGerritConfigurationServers := make(map[string]sdk.VCSGerritConfiguration)

		if _, _, err := services.NewClient(api.mustDB(), srvs).DoJSONRequest(ctx, "GET", "/vcsgerrit", nil, &vcsGerritConfigurationServers); err != nil {
			return err
		}
		// end deprecated vcs

		vcsGerritProjects, err := vcs.LoadAllVCSGerrit(ctx, api.mustDB(), gorpmapping.GetOptions.WithDecryption)
		if err != nil {
			return err
		}

		for _, v := range vcsGerritProjects {
			vcsGerritConfigurationServers[v.Name] = sdk.VCSGerritConfiguration{
				Username:      v.Auth["username"],
				SSHPrivateKey: v.Auth["sshPrivateKey"],
				URL:           v.URL,
				SSHPort:       v.Options.GerritSSHPort,
			}
		}

		return service.WriteJSON(w, vcsGerritConfigurationServers, http.StatusOK)
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
			DefaultRunRetentionPolicy: api.Config.Workflow.DefaultRetentionPolicy,
		}, http.StatusOK)
	}
}
