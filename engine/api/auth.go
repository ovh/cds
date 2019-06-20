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

func (api *API) getAuthDriversHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		drivers := []sdk.AuthDriverManifest{}

		for _, d := range api.AuthenticationDrivers {
			drivers = append(drivers, d.GetManifest())
		}

		return service.WriteJSON(w, drivers, http.StatusOK)
	}
}

// postAuthLocalSignupHandler create a new authentified user and a not verified consumer.
func (api *API) postAuthLocalSignupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		driver, okDriver := api.AuthenticationDrivers[sdk.ConsumerLocal]
		if !okDriver {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		localDriver := driver.(local.AuthDriver)

		// Extract and validate signup request
		var reqData sdk.AuthConsumerSigninRequest
		if err := service.UnmarshalBody(r, &reqData); err != nil {
			return err
		}
		if err := localDriver.CheckSignupRequest(reqData); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

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
		verifyToken, err := authentication.NewVerifyConsumerToken(consumer.ID)
		if err != nil {
			return err
		}

		// Insert the authentication
		if err := mail.SendMailVerifyToken(reqData["email"], newUser.Username, verifyToken,
			api.Config.URL.API+"/auth/consumer/local/verify/%s"); err != nil {
			log.Warning("api.postAuthLocalSignupHandler> Cannot send verify token email for user %s: %v", newUser.Username, err)
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, nil, http.StatusCreated)
	}
}

// postAuthLocalSigninHandler returns a new session for an existing local consumer.
func (api *API) postAuthLocalSigninHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		driver, okDriver := api.AuthenticationDrivers[sdk.ConsumerLocal]
		if !okDriver {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// Extract and validate signup request
		var reqData sdk.AuthConsumerSigninRequest
		if err := service.UnmarshalBody(r, &reqData); err != nil {
			return err
		}
		if err := driver.CheckSigninRequest(reqData); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Try to load a user in database for given username
		usr, err := user.LoadByUsername(ctx, tx, reqData["username"])
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.WithStack(sdk.ErrUnauthorized))
		}

		// Try to load a local consumer for user
		consumer, err := authentication.LoadConsumerByTypeAndUserID(ctx, tx, sdk.ConsumerLocal, usr.ID)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.WithStack(sdk.ErrUnauthorized))
		}

		// Check if local auth is active
		if verified, ok := consumer.Data["verified"]; !ok || verified != sdk.TrueString {
			return sdk.WithStack(sdk.ErrUnauthorized)
		}

		// Check given password with consumer password
		if hash, ok := consumer.Data["hash"]; !ok {
			return sdk.WithStack(sdk.ErrUnauthorized)
		} else if err := local.CompareHashAndPassword([]byte(hash), reqData["password"]); err != nil {
			return sdk.NewErrorWithStack(err, sdk.WithStack(sdk.ErrUnauthorized))
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

func (api *API) getAuthAskSigninHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		// Extract consumer type from request, is invalid or not in api drivers list return an error
		consumerType := sdk.AuthConsumerType(vars["consumerType"])
		if !consumerType.IsValid() {
			return sdk.WithStack(sdk.ErrNotFound)
		}
		driver, ok := api.AuthenticationDrivers[consumerType]
		if !ok {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		// Get the origin from request if set
		origin := FormString(r, "origin")
		if origin != "" && !(origin == "cdsctl" || origin == "ui") {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given origin value")
		}

		// Generate a new state value for the auth signin request
		state, err := authentication.NewSigninStateToken(origin)
		if err != nil {
			return err
		}

		// Redirect to the right signin page depending on the consumer type
		http.Redirect(w, r, driver.GetSigninURI(state), http.StatusTemporaryRedirect)
		return nil
	}
}

func (api *API) postAuthSigninHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		// Extract consumer type from request, is invalid or not in api drivers list return an error
		consumerType := sdk.AuthConsumerType(vars["consumerType"])
		if !consumerType.IsValid() {
			return sdk.WithStack(sdk.ErrNotFound)
		}
		driver, ok := api.AuthenticationDrivers[consumerType]
		if !ok {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		// Extract and validate signin request
		var req sdk.AuthConsumerSigninRequest
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}
		if err := driver.CheckSigninRequest(req); err != nil {
			return err
		}

		// Check if state is given and if its valid
		state, okState := req["state"]
		if !okState {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing state value")
		}
		if err := authentication.CheckSigninStateToken(state); err != nil {
			return err
		}

		// Convert code to external user info
		userInfo, err := driver.GetUserInfo(req)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Check if a consumer exists for consumer type and external user identifier
		consumer, err := authentication.LoadConsumerByTypeAndUserExternalID(ctx, tx, consumerType, userInfo.ExternalID)
		if err != nil {
			return err
		}
		if consumer == nil {
			// Check if a user already exists for external username
			u, err := user.LoadByUsername(ctx, tx, userInfo.Username)
			if err != nil {
				return err
			}
			if u != nil {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "a user already exists for external user name %s", userInfo.Username)
			}

			// Prepare new user
			newUser := sdk.AuthentifiedUser{
				Ring:         sdk.UserRingUser,
				Username:     userInfo.Username,
				Fullname:     userInfo.Fullname,
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

			// Create a new consumer for the new user
			consumer, err = authentication.NewConsumerExternal(tx, newUser.ID, consumerType, userInfo)
			if err != nil {
				return err
			}
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

		// Set a cookie with the jwt token
		http.SetCookie(w, &http.Cookie{
			Name:    jwtCookieName,
			Value:   jwt,
			Expires: session.ExpireAt,
		})

		usr, err := user.LoadByID(ctx, tx, consumer.AuthentifiedUserID)
		if err != nil {
			return err
		}

		// Prepare http response
		resp := sdk.AuthConsumerSigninResponse{
			Token: jwt,
			User:  usr,
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, resp, http.StatusOK)
	}
}
