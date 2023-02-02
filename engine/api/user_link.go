package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/link"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getUserLinksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		username := vars["permUsername"]

		u, err := user.LoadByUsername(ctx, api.mustDB(), username)
		if err != nil {
			return err
		}

		links, err := link.LoadUserLinksByUserID(ctx, api.mustDB(), u.ID)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, links, http.StatusOK)
	}
}

func (api *API) postLinkExternalUserWithCDSHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		username := vars["permUsername"]

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

		u, err := user.LoadByUsername(ctx, api.mustDB(), username)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		userLink := sdk.UserLink{
			AuthentifiedUserID: u.ID,
			Created:            time.Now(),
			Username:           userInfo.Username,
			Type:               string(consumerType),
		}
		log.Warn(ctx, "%+v", userLink)
		if err := link.Insert(ctx, tx, &userLink); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, userLink, http.StatusOK)
	}
}
