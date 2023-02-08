package api

import (
	"context"

	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/link"
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

func (api *API) postLinkExternalUserWithCDSHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		u := getUserConsumer(ctx)

		// Extract consumer type from request, is invalid or not in api drivers list return an error
		consumerType := sdk.AuthConsumerType(vars["consumerType"])
		linkDriver, has := api.LinkDrivers[consumerType]
		if !has {
			return sdk.WithStack(sdk.ErrInvalidData)
		}

		// Extract and validate signin request
		var req sdk.AuthConsumerSigninRequest
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}

		// Extract and validate signin state
		switch x := linkDriver.GetDriver().(type) {
		case sdk.DriverWithSignInRequest:
			if err := x.CheckSigninRequest(req); err != nil {
				return err
			}
		default:
			return sdk.WithStack(sdk.ErrInvalidData)
		}
		switch x := linkDriver.GetDriver().(type) {
		case sdk.DriverWithSigninStateToken:
			if err := x.CheckSigninStateToken(req); err != nil {
				return err
			}
		default:
			sdk.WithStack(sdk.ErrInvalidData)
		}

		// Convert code to external user info
		userInfo, err := linkDriver.GetUserInfo(ctx, req)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		userLink := sdk.UserLink{
			AuthentifiedUserID: u.AuthConsumerUser.AuthentifiedUserID,
			Created:            time.Now(),
			Username:           userInfo.Username,
			ExternalID:         userInfo.ExternalID,
			Type:               string(consumerType),
		}
		if err := link.Insert(ctx, tx, &userLink); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, userLink, http.StatusOK)
	}
}
