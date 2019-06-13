package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// postAuthLocalSignupHandler create a new authentified user and a consumer.
func (api *API) postAuthLocalSignupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		localDriver, okDriver := api.AuthenticationDrivers[sdk.ConsumerLocal]
		if !okDriver {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Extract and validate signup request
		var reqData sdk.AuthDriverRequest
		if err := service.UnmarshalBody(r, &reqData); err != nil {
			return err
		}
		if err := localDriver.CheckRequest(reqData); err != nil {
			return err
		}

		// Check that user don't already exists in database
		existingUser, err := user.LoadByUsername(ctx, tx, reqData["username"])
		if err != nil {
			return err
		}
		if existingUser != nil {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot create a user with given username")
		}

		// Prepare new user
		newUser := sdk.AuthentifiedUser{
			Ring:         sdk.UserRingUser,
			Username:     reqData["username"],
			Fullname:     reqData["fullname"],
			DateCreation: time.Now(),
		}

		// The first user is set as ADMIN
		countUsers, err := user.Count(tx)
		if err != nil {
			return err
		}
		if countUsers == 0 {
			newUser.Ring = sdk.UserRingAdmin
		}

		// Insert the new user in database
		if err := user.Insert(tx, &newUser); err != nil {
			return err
		}

		// Generate password hash to store in consumer
		hash, err := local.HashPassword(reqData["password"])
		if err != nil {
			return err
		}

		// Create new local consumer for new user, set this consumer as pending validation
		consumer, err := authentication.NewConsumerLocal(tx, newUser.ID, hash)
		if err != nil {
			return err
		}

		// Generate a token to verify consumer
		verifyToken, err := local.GenerateVerifyToken(consumer.ID)
		if err != nil {
			return err
		}

		// Insert the authentication
		if err := mail.SendMailVerifyToken(reqData["email"], newUser.Username, verifyToken,
			api.Config.URL.API+"/auth/consumer/local/verify/%s"); err != nil {
			log.Warning("api.postAuthLocalSignupHandler> Cannot send verify token email for user %s: %v", newUser.Username, err)
			return err
		}

		createUserResponse := sdk.AuthConsumerLocalSignupResponse{
			VerifyToken: verifyToken,
		}

		return service.WriteJSON(w, createUserResponse, http.StatusCreated)
	}
}

// ConfirmUser verify token send via email and mark user as verified
func (api *API) getVerifyAuthLocalHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		token := vars["token"]

		if token == "" {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		//TODO: check if has local auth

		//TODO: Verify token (as a JWT token)

		//TODO: store the new password on the local auth

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

// postAuthLocalSigninHandler returns a new session for an existing local consumer.
func (api *API) postAuthLocalSigninHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//localDriver, okDriver := api.AuthenticationDrivers[sdk.ConsumerLocal]
		//if !okDriver {
		//	return sdk.WithStack(sdk.ErrForbidden)
		//}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Extract and validate signup request
		var reqData sdk.AuthConsumerLocalSigninRequest
		if err := service.UnmarshalBody(r, &reqData); err != nil {
			return err
		}
		if err := reqData.IsValid(); err != nil {
			return err
		}

		// Try to load a user in database for given username
		usr, err := user.LoadByUsername(ctx, tx, reqData.Username)
		if err != nil {
			return err
		}
		if usr == nil {
			return sdk.WithStack(sdk.ErrUnauthorized)
		}

		// Try to load a local consumer for user
		consumer, err := authentication.LoadConsumerByTypeAndUserID(ctx, tx, sdk.ConsumerLocal, usr.ID)
		if err != nil {
			return err
		}
		if consumer == nil {
			return sdk.WithStack(sdk.ErrUnauthorized)
		}

		// Check if local auth is active
		if verified, ok := consumer.Data["verified"]; !ok || verified != sdk.TrueString {
			return sdk.WithStack(sdk.ErrUnauthorized)
		}

		// Check given password with consumer password
		hash, okHash := consumer.Data["hash"]
		if !okHash {
			return sdk.WithStack(sdk.ErrUnauthorized)
		}
		if err := local.CompareHashAndPassword([]byte(hash), reqData.Password); err != nil {
			return err
		}

		// Generate a new session for consumer
		session, err := authentication.NewSession(tx, consumer, time.Now().Add(24*time.Hour*30))
		if err != nil {
			return err
		}

		// Generate a jwt for current session
		jwt, err := authentication.NewSessionJWT(session)
		if err != nil {
			return err
		}

		// Set a cookie with the jwt token
		http.SetCookie(w, &http.Cookie{
			Name:    jwtCookieName,
			Value:   jwt,
			Expires: session.ExpireAt,
		})

		// Prepare http response
		resp := sdk.AuthConsumerLocalSigninResponse{
			Token: jwt,
			User:  usr,
		}

		return service.WriteJSON(w, resp, http.StatusOK)
	}
}

func (api *API) resetUserHandler() service.Handler {
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
}
