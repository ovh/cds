package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getLoginUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		res := sdk.UserLoginDriverResponse{}
		if api.Config.Auth.Local.Enable {
			res.Local.Available = true
		}
		return service.WriteJSON(w, res, http.StatusOK)
	}
}

func (api *API) redirectToIdentityProvider() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		http.Redirect(w, r, "", http.StatusTemporaryRedirect)
		return nil
	}
}

// LoginUser take user credentials and creates a auth token
func (api *API) postLoginUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		loginUserRequest := sdk.UserLoginRequest{}
		if err := UnmarshalBody(r, &loginUserRequest); err != nil {
			return err
		}

		// Authentify user through authDriver
		authentifier, ok := api.Router.AuthDriver.(auth.Authentifier)
		if !ok {
			return sdk.ErrMethodNotAllowed
		}

		authOK, erra := authentifier.Authentify(loginUserRequest.Username, loginUserRequest.Password)
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
		// Check default group, if any
		if err := group.CheckUserInDefaultGroup(api.mustDB(), u.ID); err != nil {
			log.Warning("Auth> Error while check user in default group:%s\n", err)
		}

		// Build a JWT token
		claims := api.claimsForUser(u)
		token, err := api.newToken(claims)

		if err != nil {
			return sdk.WrapError(err, "unable to build JWT")
		}

		// Prepare response
		response := sdk.UserAPIResponse{
			User:  *u,
			Token: token,
		}

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
