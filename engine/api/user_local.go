package api

import (
	"context"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/authentication/localauthentication"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk/log"
)

// postAuthLocalSignupHandler create a new authentified user and a consumer.
func (api *API) postAuthLocalSignupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// TODO: check if local auth is activated
		tx, err := api.mustDB().Begin()
		if err != nil {
			return err
		}

		defer tx.Rollback() // nolint

		createUserRequest := sdk.UserRequest{}
		if err := service.UnmarshalBody(r, &createUserRequest); err != nil {
			return err
		}

		if createUserRequest.Username == "" {
			return sdk.WrapError(sdk.ErrInvalidUsername, "AddUser: Empty username is invalid")
		}

		if createUserRequest.Password != createUserRequest.PasswordConfirmation {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		//	if !user.IsValidEmail(createUserRequest.Email) {
		//		return sdk.WrapError(sdk.ErrInvalidEmail, "AddUser: Email address %s is not valid", createUserRequest.Email)
		//	}

		//if !user.IsAllowedDomain(api.Config.Auth.Local.SignupAllowedDomains, createUserRequest.Email) {
		//	return sdk.WrapError(sdk.ErrInvalidEmailDomain, "AddUser: Email address %s does not have a valid domain. Allowed domains:%v", createUserRequest.Email, api.Config.Auth.Local.SignupAllowedDomains)
		//}

		if err := localauthentication.CheckPasswordIsValid(createUserRequest.Password); err != nil {
			return err
		}

		verifyToken, err := localauthentication.GenerateVerifyToken(createUserRequest.Username)
		if err != nil {
			return err
		}

		var usr = sdk.AuthentifiedUser{
			Ring:         sdk.UserRingUser,
			Username:     createUserRequest.Username,
			Fullname:     createUserRequest.Fullname,
			DateCreation: time.Now(),
		}

		// The first user is set as ADMIN
		nbUsers, errc := user.CountUser(api.mustDB())
		if errc != nil {
			return sdk.WrapError(errc, "AddUser: Cannot count user")
		}
		if nbUsers == 0 {
			usr.Ring = sdk.UserRingAdmin
		}

		// Insert the user
		if err := user.Insert(tx, &usr); err != nil {
			return err
		}

		// Insert the authentication
		localAuth := sdk.UserLocalAuthentication{
			UserID:        usr.ID,
			ClearPassword: createUserRequest.Password,
		}

		if err := localauthentication.Insert(tx, &localAuth); err != nil {
			return err
		}

		if err := mail.SendMailVerifyToken(createUserRequest.Email, createUserRequest.Username, verifyToken, createUserRequest.Callback); err != nil {
			log.Warning("addUserHandler.SendMailVerifyToken> Cannot send verify token email for user %s : %v", createUserRequest.Username, err)
			return err
		}

		createUserResponse := sdk.UserResponse{AuthentifiedUser: usr, VerifyToken: verifyToken}

		return service.WriteJSON(w, createUserResponse, http.StatusCreated)
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

		verifyToken, err := localauthentication.GenerateVerifyToken(resetUserRequest.Username)
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

// ConfirmUser verify token send via email and mark user as verified
func (api *API) confirmUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get user name in URL
		vars := mux.Vars(r)
		username := vars["username"]
		token := vars["token"]

		if username == "" || token == "" {
			return sdk.ErrInvalidUsername
		}

		// Load user
		usr, err := user.LoadByUsername(ctx, api.mustDB(), username)
		if err != nil {
			return err
		}

		_ = usr

		//TODO: check if has local auth

		//TODO: Verify token (as a JWT token)

		//TODO: store the new password on the local auth

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}
