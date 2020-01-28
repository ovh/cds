package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getAuthDriversHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		drivers := make(sdk.AuthDriverManifests, 0, len(api.AuthenticationDrivers))

		for _, d := range api.AuthenticationDrivers {
			drivers = append(drivers, d.GetManifest())
		}

		countAdmins, err := user.CountAdmin(api.mustDB())
		if err != nil {
			return err
		}

		var response = sdk.AuthDriverResponse{
			IsFirstConnection: countAdmins == 0,
			Drivers:           drivers,
		}

		return service.WriteJSON(w, response, http.StatusOK)
	}
}

func (api *API) getAuthScopesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, sdk.AuthConsumerScopes, http.StatusOK)
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

		driverRedirect, ok := driver.(sdk.AuthDriverWithRedirect)
		if !ok {
			return nil
		}

		countAdmins, err := user.CountAdmin(api.mustDB())
		if err != nil {
			return err
		}

		var signinState = sdk.AuthSigninConsumerToken{
			RequireMFA:        QueryBool(r, "require_mfa"),
			RedirectURI:       QueryString(r, "redirect_uri"),
			IsFirstConnection: countAdmins == 0,
		}
		// Get the origin from request if set
		signinState.Origin = QueryString(r, "origin")

		if signinState.Origin != "" && !(signinState.Origin == "cdsctl" || signinState.Origin == "ui") {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given origin value")
		}

		// Redirect to the right signin page depending on the consumer type
		redirect, err := driverRedirect.GetSigninURI(signinState)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, redirect, http.StatusOK)
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

		// Extract and validate signin state
		switch x := driver.(type) {
		case sdk.AuthDriverWithSigninStateToken:
			if err := x.CheckSigninStateToken(req); err != nil {
				return err
			}
		}

		// Convert code to external user info
		userInfo, err := driver.GetUserInfo(ctx, req)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		var signupDone bool
		initToken, hasInitToken := req["init_token"]
		hasInitToken = hasInitToken && initToken != ""

		// Check if a consumer exists for consumer type and external user identifier
		consumer, err := authentication.LoadConsumerByTypeAndUserExternalID(ctx, tx, consumerType, userInfo.ExternalID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}

		// If there is no existing consumer we should check if a user need to be created
		// Then we want to create a new consumer for current type
		if consumer == nil {
			var u *sdk.AuthentifiedUser

			currentConsumer := getAPIConsumer(ctx)
			if currentConsumer != nil {
				// If no consumer already exists for given request, but there is a current session
				// We should create a new consumer for the current consumer's user
				u = currentConsumer.AuthentifiedUser

				// If new consumer email not already on exiting user add it
				existingContact, err := user.LoadContactByTypeAndValue(ctx, tx, sdk.UserContactTypeEmail, userInfo.Email)
				if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
					return err
				}
				if existingContact == nil {
					// Insert a secondary contact for the existing user in database
					if err := user.InsertContact(ctx, tx, &sdk.UserContact{
						Primary:  false,
						Type:     sdk.UserContactTypeEmail,
						UserID:   u.ID,
						Value:    userInfo.Email,
						Verified: true,
					}); err != nil {
						return err
					}
				}
				if err := group.CheckUserInDefaultGroup(ctx, tx, u.ID); err != nil {
					return err
				}
			} else {
				// Check if a user already exists for external username
				u, err = user.LoadByUsername(ctx, tx, userInfo.Username, user.LoadOptions.WithContacts)
				if err != nil && !sdk.ErrorIs(err, sdk.ErrUserNotFound) {
					return err
				}
				if u != nil {
					// If the user exists with the same email address than in the userInfo,
					// we will create a new consumer and continue the signin
					// else we raise an error
					if u.GetEmail() != userInfo.Email {
						return sdk.NewErrorFrom(sdk.ErrForbidden, "a user already exists for username %s", userInfo.Username)
					}
				} else {
					// Check if a user already exists for external email
					contact, err := user.LoadContactByTypeAndValue(ctx, tx, sdk.UserContactTypeEmail, userInfo.Email)
					if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
						return err
					}
					if contact != nil {
						// A user already exists with an other username but the same email address
						u, err = user.LoadByID(ctx, tx, contact.UserID, user.LoadOptions.WithContacts)
						if err != nil {
							return err
						}
					} else {
						// We can't find any user with the same email address
						// So we will do signup for a new user from the data got from the auth driver
						if driver.GetManifest().SignupDisabled {
							return sdk.WithStack(sdk.ErrSignupDisabled)
						}

						u = &sdk.AuthentifiedUser{
							Ring:     sdk.UserRingUser,
							Username: userInfo.Username,
							Fullname: userInfo.Fullname,
						}

						// If a magic token is given and there is no admin already registered, set new user as admin
						countAdmins, err := user.CountAdmin(tx)
						if err != nil {
							return err
						}
						if countAdmins == 0 && hasInitToken {
							u.Ring = sdk.UserRingAdmin
						} else {
							hasInitToken = false
						}

						// Insert the new user in database
						if err := user.Insert(ctx, tx, u); err != nil {
							return err
						}

						// Insert the primary contact for the new user in database
						if err := user.InsertContact(ctx, tx, &sdk.UserContact{
							Primary:  true,
							Type:     sdk.UserContactTypeEmail,
							UserID:   u.ID,
							Value:    userInfo.Email,
							Verified: true,
						}); err != nil {
							return err
						}

						if err := group.CheckUserInDefaultGroup(ctx, tx, u.ID); err != nil {
							return err
						}

						signupDone = true
					}
				}
			}

			// Create a new consumer for the new user
			consumer, err = authentication.NewConsumerExternal(ctx, tx, u.ID, consumerType, userInfo)
			if err != nil {
				return err
			}
		}

		// If a new user has been created and a first admin has been create,
		// let's init the builtin consumers from the magix token
		if signupDone && hasInitToken {
			if err := initBuiltinConsumersFromStartupConfig(ctx, tx, consumer, initToken); err != nil {
				return err
			}
		}

		// Generate a new session for consumer
		session, err := authentication.NewSession(ctx, tx, consumer, driver.GetSessionDuration(), userInfo.MFA)
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
		api.SetCookie(w, jwtCookieName, jwt, session.ExpireAt)

		// Prepare http response
		resp := sdk.AuthConsumerSigninResponse{
			Token:  jwt,
			User:   usr,
			APIURL: api.Config.URL.API,
		}

		return service.WriteJSON(w, resp, http.StatusOK)
	}
}

func (api *API) postAuthSignoutHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		session := getAuthSession(ctx)

		if err := authentication.DeleteSessionByID(api.mustDB(), session.ID); err != nil {
			return err
		}

		// Delete the jwt cookie value
		api.UnsetCookie(w, jwtCookieName)

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) postAuthDetachHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		// Extract consumer type from request, is invalid or not in api drivers list return an error
		consumerType := sdk.AuthConsumerType(vars["consumerType"])
		if !consumerType.IsValidExternal() {
			return sdk.WithStack(sdk.ErrNotFound)
		}
		_, ok := api.AuthenticationDrivers[consumerType]
		if !ok {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		currentConsumer := getAPIConsumer(ctx)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		consumer, err := authentication.LoadConsumerByTypeAndUserID(ctx, tx, consumerType, currentConsumer.AuthentifiedUserID)
		if err != nil {
			return err
		}

		if err := authentication.DeleteConsumerByID(tx, consumer.ID); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		// If we just removed the current consumer, clean http cookie.
		if consumer.ID == currentConsumer.ID {
			api.UnsetCookie(w, jwtCookieName)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) getAuthMe() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		c := getAPIConsumer(ctx)
		s := getAuthSession(ctx)
		if c == nil || s == nil {
			return sdk.WithStack(sdk.ErrUnauthorized)
		}
		return service.WriteJSON(w, sdk.AuthCurrentConsumerResponse{
			Consumer: *c,
			Session:  *s,
		}, http.StatusOK)
	}
}
