package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
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
		consumer, err := authentication.LoadConsumerByID(ctx, tx, userInfo.ExternalID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
		if err != nil {
			return sdk.NewError(sdk.ErrForbidden, err)
		}

		token, err := req.StringE("token")
		if err != nil {
			return err
		}

		// Check the Token validity againts the IAT attribute
		if _, err := builtin.CheckSigninConsumerTokenIssuedAt(ctx, token, consumer); err != nil {
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

		// Set those values (has it would be done in api.authOptionalMiddleware)
		ctx = context.WithValue(ctx, contextConsumer, consumer)
		ctx = context.WithValue(ctx, cdslog.AuthUserID, consumer.AuthentifiedUserID)
		SetTracker(w, cdslog.AuthUserID, consumer.AuthentifiedUserID)
		ctx = context.WithValue(ctx, cdslog.AuthConsumerID, consumer.ID)
		SetTracker(w, cdslog.AuthConsumerID, consumer.ID)

		ctx = context.WithValue(ctx, contextSession, session)
		ctx = context.WithValue(ctx, cdslog.AuthSessionID, session.ID)
		SetTracker(w, cdslog.AuthSessionID, session.ID)
		ctx = context.WithValue(ctx, cdslog.AuthSessionIAT, session.Created.Unix())
		SetTracker(w, cdslog.AuthSessionIAT, session.Created.Unix())
		ctx = context.WithValue(ctx, contextSession, session)

		// If the Signin has a *service* Payload, we have to perform the service registration
		srvInput, has := req["service"]
		var srv sdk.Service
		if has {
			btes, err := json.Marshal(srvInput)
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}

			if err := sdk.JSONUnmarshal(btes, &srv); err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}

			ctx = context.WithValue(ctx, cdslog.AuthServiceName, srv.Name)
			SetTracker(w, cdslog.AuthServiceName, srv.Name)

			if err := api.serviceRegister(ctx, tx, &srv); err != nil {
				return err
			}
		} else {
			ctx = context.WithValue(ctx, cdslog.AuthUsername, consumer.AuthentifiedUser.Username)
			SetTracker(w, cdslog.AuthUsername, consumer.AuthentifiedUser.Username)
		}

		// Set a cookie with the jwt token
		api.SetCookie(w, service.JWTCookieName, jwt, session.ExpireAt, true)

		usr, err := user.LoadByID(ctx, tx, consumer.AuthentifiedUserID)
		if err != nil {
			return err
		}

		// Prepare http response
		resp := sdk.AuthConsumerSigninResponse{
			Token:   jwt,
			User:    usr,
			APIURL:  api.Config.URL.API,
			Service: &srv,
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
