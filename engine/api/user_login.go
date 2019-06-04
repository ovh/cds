package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/user"

	"github.com/ovh/cds/engine/api/accesstoken"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/log"
)

// LoginUser take user credentials and creates a auth token
func (api *API) loginUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var loginUserRequest sdk.UserLoginRequest
		if err := service.UnmarshalBody(r, &loginUserRequest); err != nil {
			return err
		}

		//var logFromCLI bool
		//if r.Header.Get(sdk.RequestedWithHeader) == sdk.RequestedWithValue {
		//	log.Info("LoginUser> login from CLI")
		//	logFromCLI = true
		//}

		var authUser *sdk.AuthentifiedUser
		for authName, authDriver := range api.AuthenticationDrivers {
			var err error
			authUser, err = authDriver.CheckAuthentication(ctx, api.mustDB(), r)
			if err != nil {
				log.Warning("loginUserHandler> %s login failed: %v", authName, err)
				if sdk.ErrorIs(err, sdk.ErrUnauthorized) {
					continue
				}
				return err
			}
		}

		if authUser == nil {
			return sdk.WithStack(sdk.ErrUnauthorized)
		}

		// If there is a request token, store it (for 30 minutes)
		if loginUserRequest.RequestToken != "" {
			var accessTokenRequest sdk.AccessTokenRequest
			if err := jws.UnsafeParse(loginUserRequest.RequestToken, &accessTokenRequest); err != nil {
				return sdk.WithStack(err)
			}
			token, _, err := api.createNewAccessToken(*authUser, accessTokenRequest)
			if err != nil {
				return sdk.WithStack(err)
			}
			api.Cache.SetWithTTL("api:loginUserHandler:RequestToken:"+loginUserRequest.RequestToken, token, 30*60)
		}

		if err := user.LoadOptions.WithDeprecatedUser(ctx, api.mustDB(), authUser); err != nil {
			return err
		}

		if err := group.CheckUserInDefaultGroup(api.mustDB(), authUser.OldUserStruct.ID); err != nil {
			log.Warning("Auth> Error while check user in default group:%s\n", err)
		}

		var loginUserResponse sdk.UserLoginResponse
		return service.WriteJSON(w, loginUserResponse, http.StatusOK)
	}
}

func (api *API) loginUserCallbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// api.Cache.SetWithTTL("api:loginUserHandler:RequestToken:"+loginUserRequest.RequestToken, token, 30*60)
		var request sdk.UserLoginCallbackRequest
		if err := service.UnmarshalBody(r, &request); err != nil {
			return sdk.WithStack(err)
		}

		var accessToken sdk.AccessToken
		if !api.Cache.Get("api:loginUserHandler:RequestToken:"+request.RequestToken, &accessToken) {
			return sdk.ErrNotFound
		}

		pk, err := jws.NewPublicKeyFromPEM(request.PublicKey)
		if err != nil {
			log.Debug("unable to read public key: %v", string(request.PublicKey))
			return sdk.WithStack(err)
		}

		var x sdk.AccessTokenRequest
		if err := jws.Verify(pk, request.RequestToken, &x); err != nil {
			return sdk.WithStack(err)
		}

		jwt, err := accesstoken.Regen(&accessToken)
		if err != nil {
			return sdk.WithStack(err)
		}

		w.Header().Add("X-CDS-JWT", jwt)

		return service.WriteJSON(w, accessToken, http.StatusOK)
	}
}
