package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getLinkDriversHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ds := make([]string, 0, len(api.LinkDrivers))
		for k := range api.LinkDrivers {
			ds = append(ds, string(k))
		}
		return service.WriteJSON(w, ds, http.StatusOK)
	}
}

func (api *API) postAskLinkExternalUserWithCDSHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		// Extract consumer type from request, is invalid or not in api drivers list return an error
		consumerType := sdk.AuthConsumerType(vars["consumerType"])

		linkDriver, has := api.LinkDrivers[consumerType]
		if !has {
			return sdk.WithStack(sdk.ErrNotImplemented)
		}

		driverRedirect, ok := linkDriver.GetDriver().(sdk.DriverWithRedirect)
		if !ok {
			return nil
		}

		var signinState = sdk.AuthSigninConsumerToken{
			RedirectURI: QueryString(r, "redirect_uri"),
		}
		// Get the origin from request if set
		signinState.Origin = QueryString(r, "origin")

		if signinState.Origin != "" && !(signinState.Origin == "cdsctl" || signinState.Origin == "ui") {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given origin value")
		}
		signinState.LinkUser = true

		// Redirect to the right signin page depending on the consumer type
		redirect, err := driverRedirect.GetSigninURI(signinState)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, redirect, http.StatusOK)
	}
}
