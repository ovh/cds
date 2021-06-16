package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
)

func (api *API) postAuthBuiltinSigninHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get the consumer builtin driver
		driver, ok := api.AuthenticationDrivers[sdk.ConsumerBuiltin]
		if !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// Extract and validate signin request
		var req sdk.AuthConsumerSigninRequest
		if err := service.UnmarshalBody(r, &req); err != nil {
			return sdk.NewError(sdk.ErrForbidden, err)
		}
		if err := driver.CheckSigninRequest(req); err != nil {
			return sdk.NewError(sdk.ErrForbidden, err)
		}
		// Convert code to external user info
		userInfo, err := driver.GetUserInfo(ctx, req)
		if err != nil {
			return sdk.NewError(sdk.ErrForbidden, err)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.NewError(sdk.ErrForbidden, err)
		}
		defer tx.Rollback() // nolint

		// Check if a consumer exists for consumer type and external user identifier
		consumer, err := authentication.LoadConsumerByID(ctx, tx, userInfo.ExternalID)
		if err != nil {
			return sdk.NewError(sdk.ErrForbidden, err)
		}

		// Check the Token validity againts the IAT attribute
		if _, err := builtin.CheckSigninConsumerTokenIssuedAt(ctx, req["token"], consumer); err != nil {
			return sdk.NewError(sdk.ErrForbidden, err)
		}

		// Generate a new session for consumer
		session, err := authentication.NewSession(ctx, tx, consumer, driver.GetSessionDuration())
		if err != nil {
			return err
		}

		// Store the last authentication date on the consumer
		now := time.Now()
		consumer.LastAuthentication = &now
		if err := authentication.UpdateConsumerLastAuthentication(ctx, tx, consumer); err != nil {
			return err
		}

		// Generate a jwt for current session
		jwt, err := authentication.NewSessionJWT(session, "")
		if err != nil {
			return err
		}

		// Set a cookie with the jwt token
		api.SetCookie(w, service.JWTCookieName, jwt, session.ExpireAt, true)

		usr, err := user.LoadByID(ctx, tx, consumer.AuthentifiedUserID)
		if err != nil {
			return err
		}

		// Prepare http response
		resp := sdk.AuthConsumerSigninResponse{
			Token:  jwt,
			User:   usr,
			APIURL: api.Config.URL.API,
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		pubKey, err := jws.ExportPublicKey(authentication.GetSigningKey())
		if err != nil {
			return sdk.WrapError(err, "Unable to export public signing key")
		}

		encodedPubKey := base64.StdEncoding.EncodeToString(pubKey)
		w.Header().Set("X-Api-Pub-Signing-Key", encodedPubKey)

		return service.WriteJSON(w, resp, http.StatusOK)
	}
}
