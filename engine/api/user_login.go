package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoginUser take user credentials and creates a auth token
func (api *API) loginUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		loginUserRequest := sdk.UserLoginRequest{}
		if err := UnmarshalBody(r, &loginUserRequest); err != nil {
			return err
		}

		var logFromCLI bool
		if r.Header.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
			log.Info("LoginUser> login from CLI")
			logFromCLI = true
		}

		// Authentify user through authDriver
		authOK, erra := api.Router.AuthDriver.Authentify(loginUserRequest.Username, loginUserRequest.Password)
		if erra != nil {
			return sdk.WrapError(sdk.ErrInvalidUser, "Auth> Login error %s: %s", loginUserRequest.Username, erra)
		}
		if !authOK {
			return sdk.WrapError(sdk.ErrInvalidUser, "Auth> Login failed: %s", loginUserRequest.Username)
		}
		// Load user
		u, errl := user.LoadUserWithoutAuth(api.mustDB(), loginUserRequest.Username)
		if errl != nil && errl == sql.ErrNoRows {
			return sdk.WrapError(sdk.ErrInvalidUser, "Auth> Login error %s: %s", loginUserRequest.Username, errl)
		}
		if errl != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "Auth> Login error %s: %s", loginUserRequest.Username, errl)
		}

		// Prepare response
		response := sdk.UserAPIResponse{
			User: *u,
		}

		if err := group.CheckUserInDefaultGroup(api.mustDB(), u.ID); err != nil {
			log.Warning("Auth> Error while check user in default group:%s\n", err)
		}

		var sessionKey sessionstore.SessionKey
		var errs error
		if !logFromCLI {
			//Standard login, new session
			sessionKey, errs = auth.NewSession(api.Router.AuthDriver, u)
			if errs != nil {
				log.Error("Auth> Error while creating new session: %s\n", errs)
			}
		} else {
			//CLI login, generate user key as persistent session
			sessionKey, errs = auth.NewPersistentSession(api.mustDB(), api.Router.AuthDriver, u)
			if errs != nil {
				log.Error("Auth> Error while creating new session: %s\n", errs)
			}
		}

		if sessionKey != "" {
			w.Header().Set(sdk.SessionTokenHeader, string(sessionKey))
			response.Token = string(sessionKey)
		}

		response.User.Auth = sdk.Auth{}
		response.User.Permissions = sdk.UserPermissions{}
		return service.WriteJSON(w, response, http.StatusOK)
	}
}

func (api *API) postRequestTokenHanler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Handle the payload
		// It can be a token request by a User, and Hatchery,
		// a Worker (for queue polling, or working on a job or a Service
		//
		//	{
		//		"audience": "user|worker|worker_polling|service|hatchery|service",
		//		"username": "",
		//		"password": "",
		//		"token": "", 			// Can be an API Token or a SAML token
		//	}
		//
		// As a result, it returns {token: <token>} which must be used with a Authorisation: Bearer <token>
		return nil
	}
}
