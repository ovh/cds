package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// postAuthLocalSignupHandler creates a new registration that need to be verified to create a new user.
func (api *API) postAuthLocalSignupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		driver, okDriver := api.AuthenticationDrivers[sdk.ConsumerLocal]
		if !okDriver || driver.GetManifest().SignupDisabled {
			return sdk.WithStack(sdk.ErrSignupDisabled)
		}

		localDriver := driver.(*local.AuthDriver)

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
		if err != nil && !sdk.ErrorIs(err, sdk.ErrUserNotFound) {
			return err
		}
		if existingUser != nil {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot create a user with given username")
		}

		// Check that user contact don't already exists in database for given email
		existingEmail, err := user.LoadContactByTypeAndValue(ctx, tx, sdk.UserContactTypeEmail, reqData["email"])
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if existingEmail != nil {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot create a user with given email")
		}

		// Generate password hash to store in consumer
		hash, err := local.HashPassword(reqData["password"])
		if err != nil {
			return err
		}

		// Prepare new user registration
		reg := sdk.UserRegistration{
			Username: reqData["username"],
			Fullname: reqData["fullname"],
			Email:    reqData["email"],
			Hash:     string(hash),
		}

		// Insert the new user in database
		if err := local.InsertRegistration(ctx, tx, &reg); err != nil {
			return err
		}

		countAdmins, err := user.CountAdmin(tx)
		if err != nil {
			return err
		}

		// Generate a token to verify registration
		regToken, err := local.NewRegistrationToken(api.Cache, reg.ID, countAdmins == 0)
		if err != nil {
			return err
		}

		// Insert the authentication
		if err := mail.SendMailVerifyToken(ctx, reg.Email, reg.Username, regToken,
			api.Config.URL.UI+"/auth/verify?token=%s",
			api.Config.URL.API,
		); err != nil {
			return sdk.WrapError(err, "cannot send verify token email for user %s", reg.Username)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, nil, http.StatusCreated)
	}
}

func initBuiltinConsumersFromStartupConfig(ctx context.Context, tx gorp.SqlExecutor, consumer *sdk.AuthConsumer, initToken string) error {
	// Deserialize the magic token to retrieve the startup configuration
	var startupConfig StartupConfig
	if err := authentication.VerifyJWS(initToken, &startupConfig); err != nil {
		return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given init token"))
	}

	log.Warning(ctx, "Magic token detected !: %s", initToken)

	if startupConfig.IAT == 0 || startupConfig.IAT > time.Now().Unix() {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given init token, issued at value should be set and can not be in the future")
	}

	// Create the consumers provided by the startup configuration
	for _, cfg := range startupConfig.Consumers {
		var scopes sdk.AuthConsumerScopeSlice

		switch cfg.ServiceType {
		case services.TypeHatchery:
			scopes = []sdk.AuthConsumerScope{sdk.AuthConsumerScopeService, sdk.AuthConsumerScopeHatchery, sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeWorkerModel}
		case services.TypeHooks:
			scopes = []sdk.AuthConsumerScope{sdk.AuthConsumerScopeService, sdk.AuthConsumerScopeHooks, sdk.AuthConsumerScopeProject, sdk.AuthConsumerScopeRun}
		default:
			scopes = []sdk.AuthConsumerScope{sdk.AuthConsumerScopeService}
		}

		var c = sdk.AuthConsumer{
			ID:                 cfg.ID,
			Name:               cfg.Name,
			Description:        cfg.Description,
			AuthentifiedUserID: consumer.AuthentifiedUserID,
			ParentID:           &consumer.ID,
			Type:               sdk.ConsumerBuiltin,
			Data:               map[string]string{},
			GroupIDs:           []int64{group.SharedInfraGroup.ID},
			Scopes:             scopes,
			IssuedAt:           time.Unix(startupConfig.IAT, 0),
		}

		if err := authentication.InsertConsumer(ctx, tx, &c); err != nil {
			return err
		}
	}

	return nil
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
			return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}

		// Try to load a local consumer for user
		consumer, err := authentication.LoadConsumerByTypeAndUserID(ctx, tx, sdk.ConsumerLocal, usr.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}

		// Check given password with consumer password
		if hash, ok := consumer.Data["hash"]; !ok {
			return sdk.WithStack(sdk.ErrUnauthorized)
		} else if err := local.CompareHashAndPassword([]byte(hash), reqData["password"]); err != nil {
			return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}

		// Generate a new session for consumer
		session, err := authentication.NewSession(ctx, tx, consumer, driver.GetSessionDuration(), false)
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

func (api *API) postAuthLocalVerifyHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		driver, okDriver := api.AuthenticationDrivers[sdk.ConsumerLocal]
		if !okDriver {
			return sdk.WithStack(sdk.ErrForbidden)
		}
		localDriver := driver.(*local.AuthDriver)

		var reqData sdk.AuthConsumerSigninRequest
		var tokenInQueryString = QueryString(r, "token")
		if tokenInQueryString != "" {
			reqData["token"] = tokenInQueryString
		} else {
			if err := service.UnmarshalBody(r, &reqData); err != nil {
				return err
			}
		}
		if err := localDriver.CheckVerifyRequest(reqData); err != nil {
			return err
		}

		initToken, hasInitToken := reqData["init_token"]
		hasInitToken = hasInitToken && initToken != ""

		registrationID, err := local.CheckRegistrationToken(api.Cache, reqData["token"])
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Get the registration entry from database then delete it
		reg, err := local.LoadRegistrationByID(ctx, tx, registrationID)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}
		if err := local.DeleteRegistrationByID(tx, reg.ID); err != nil {
			return err
		}

		// Prepare new user
		newUser := sdk.AuthentifiedUser{
			Ring:     sdk.UserRingUser,
			Username: reg.Username,
			Fullname: reg.Fullname,
		}

		countAdmins, err := user.CountAdmin(tx)
		if err != nil {
			return err
		}
		// If a magic token is given and there is no admin already registered, set new user as admin
		// the init token validity will be checked just after
		if countAdmins == 0 && hasInitToken {
			newUser.Ring = sdk.UserRingAdmin
		}

		// Insert the new user in database
		if err := user.Insert(ctx, tx, &newUser); err != nil {
			return err
		}

		userContact := sdk.UserContact{
			Primary:  true,
			Type:     sdk.UserContactTypeEmail,
			UserID:   newUser.ID,
			Value:    reg.Email,
			Verified: true,
		}

		// Insert the primary contact for the new user in database
		if err := user.InsertContact(ctx, tx, &userContact); err != nil {
			return err
		}

		if err := group.CheckUserInDefaultGroup(ctx, tx, newUser.ID); err != nil {
			return err
		}

		// Create new local consumer for new user, set this consumer as pending validation
		consumer, err := local.NewConsumerWithHash(ctx, tx, newUser.ID, reg.Hash)
		if err != nil {
			return err
		}

		// Check if a magic init token was given for first signup
		if countAdmins == 0 && hasInitToken {
			if err := initBuiltinConsumersFromStartupConfig(ctx, tx, consumer, initToken); err != nil {
				return err
			}
		}

		// Generate a new session for consumer
		session, err := authentication.NewSession(ctx, tx, consumer, driver.GetSessionDuration(), false)
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

		local.CleanVerifyConsumerToken(api.Cache, consumer.ID)

		// Set a cookie with the jwt token
		api.SetCookie(w, jwtCookieName, jwt, session.ExpireAt)

		// Prepare http response
		resp := sdk.AuthConsumerSigninResponse{
			APIURL: api.Config.URL.API,
			Token:  jwt,
			User:   usr,
		}

		return service.WriteJSON(w, resp, http.StatusOK)
	}
}

func (api *API) postAuthLocalAskResetHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		driver, okDriver := api.AuthenticationDrivers[sdk.ConsumerLocal]
		if !okDriver {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		localDriver := driver.(*local.AuthDriver)

		var email string

		// If there is a consumer, send directly to the primary contact for the user.
		consumer := getAPIConsumer(ctx)
		if consumer != nil {
			email = consumer.GetEmail()
		} else {
			var reqData sdk.AuthConsumerSigninRequest
			if err := service.UnmarshalBody(r, &reqData); err != nil {
				return err
			}
			if err := localDriver.CheckAskResetRequest(reqData); err != nil {
				return err
			}
			email = reqData["email"]
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		contact, err := user.LoadContactByTypeAndValue(ctx, tx, sdk.UserContactTypeEmail, email)
		if err != nil {
			// If there is no contact for given email, return ok to prevent email exploration
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				log.Warning(ctx, "api.postAuthLocalAskResetHandler> no contact found for email %s: %v", email, err)
				return service.WriteJSON(w, nil, http.StatusOK)
			}
			return err
		}

		existingLocalConsumer, err := authentication.LoadConsumerByTypeAndUserID(ctx, tx, sdk.ConsumerLocal, contact.UserID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if existingLocalConsumer == nil {
			// If a user exists for given email but has no local consumer we want to create it.
			existingLocalConsumer, err = local.NewConsumer(ctx, tx, contact.UserID)
			if err != nil {
				return err
			}
		}
		if err := authentication.LoadConsumerOptions.WithAuthentifiedUser(ctx, tx, existingLocalConsumer); err != nil {
			return err
		}

		resetToken, err := local.NewResetConsumerToken(api.Cache, existingLocalConsumer.ID)
		if err != nil {
			return err
		}

		// Insert the authentication
		if err := mail.SendMailAskResetToken(ctx, contact.Value, existingLocalConsumer.AuthentifiedUser.Username, resetToken,
			api.Config.URL.UI+"/auth/reset?token=%s",
			api.Config.URL.API,
		); err != nil {
			return sdk.WrapError(err, "cannot send reset token email at %s", contact.Value)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) postAuthLocalResetHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		driver, okDriver := api.AuthenticationDrivers[sdk.ConsumerLocal]
		if !okDriver {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		localDriver := driver.(*local.AuthDriver)

		var reqData sdk.AuthConsumerSigninRequest
		if err := service.UnmarshalBody(r, &reqData); err != nil {
			return err
		}

		if reqData["token"] == "" {
			reqData["token"] = QueryString(r, "token")
		}

		if err := localDriver.CheckResetRequest(reqData); err != nil {
			return err
		}

		consumerID, err := local.CheckResetConsumerToken(api.Cache, reqData["token"])
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
			return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}
		if consumer.Type != sdk.ConsumerLocal {
			return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}

		// Generate password hash to store in consumer
		hash, err := local.HashPassword(reqData["password"])
		if err != nil {
			return err
		}

		consumer.Data["hash"] = string(hash)
		if err := authentication.UpdateConsumer(ctx, tx, consumer); err != nil {
			return err
		}

		// Generate a new session for consumer
		session, err := authentication.NewSession(ctx, tx, consumer, driver.GetSessionDuration(), false)
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

		local.CleanResetConsumerToken(api.Cache, consumer.ID)

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
