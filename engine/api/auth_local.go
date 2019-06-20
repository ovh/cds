package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getAuthLocalVerifyHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		driver, okDriver := api.AuthenticationDrivers[sdk.ConsumerLocal]
		if !okDriver {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		vars := mux.Vars(r)
		tokenString := vars["token"]

		if tokenString == "" {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		consumerID, err := authentication.CheckVerifyConsumerToken(tokenString)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Get the consumer from database and set it to verified
		consumer, err := authentication.LoadConsumerByID(ctx, tx, consumerID)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.WithStack(sdk.ErrUnauthorized))
		}
		if consumer.Type != sdk.ConsumerLocal {
			return sdk.NewErrorWithStack(err, sdk.WithStack(sdk.ErrUnauthorized))
		}

		consumer.Data["verified"] = sdk.TrueString
		if err := authentication.UpdateConsumer(tx, consumer); err != nil {
			return err
		}

		// Generate a new session for consumer
		session, err := authentication.NewSession(tx, consumer, driver.GetSessionDuration())
		if err != nil {
			return err
		}

		// Generate a jwt for current session
		jwt, err := authentication.NewSessionJWT(session)
		if err != nil {
			return err
		}

		usr, err := user.LoadByID(ctx, tx, consumer.AuthentifiedUserID)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		// Set a cookie with the jwt token
		http.SetCookie(w, &http.Cookie{
			Name:    jwtCookieName,
			Value:   jwt,
			Expires: session.ExpireAt,
		})

		// Prepare http response
		resp := sdk.AuthConsumerSigninResponse{
			Token: jwt,
			User:  usr,
		}

		return service.WriteJSON(w, resp, http.StatusOK)
	}
}

/*func (api *API) resetUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["username"]

		var resetUserRequest sdk.UserResetRequest
		if err := service.UnmarshalBody(r, &resetUserRequest); err != nil {
			return err
		}

		if resetUserRequest.Username != username {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		usr, err := user.LoadByUsername(ctx, api.mustDB(), username, user.LoadOptions.WithContacts)
		if err != nil {
			return err
		}

		// TODO: Check if user has local auth

		contact := usr.Contacts.Find(sdk.UserContactTypeEmail, resetUserRequest.Email)
		if contact == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		verifyToken, err := local.GenerateVerifyToken(resetUserRequest.Username)
		if err != nil {
			return err
		}

		if err := mail.SendMailVerifyToken(resetUserRequest.Email, resetUserRequest.Username, verifyToken, resetUserRequest.Callback); err != nil {
			log.Warning("resetUserHandler.SendMailVerifyToken> Cannot send verify token email for user %s : %v", resetUserRequest.Username, err)
			return err
		}
		resetUserResponse := sdk.UserResponse{AuthentifiedUser: *usr, VerifyToken: verifyToken}

		return service.WriteJSON(w, resetUserResponse, http.StatusOK)
	}
}*/
