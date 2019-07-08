package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// DeleteUserHandler removes a user.
func (api *API) deleteUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsername"]

		consumer := getAPIConsumer(ctx)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback()

		var u *sdk.AuthentifiedUser
		if username == "me" {
			u, err = user.LoadByID(ctx, tx, consumer.AuthentifiedUserID, user.LoadOptions.Default)
		} else {
			u, err = user.LoadByUsername(ctx, tx, username, user.LoadOptions.Default)
		}
		if err != nil {
			return err
		}

		if err := user.DeleteByID(tx, u.ID); err != nil {
			return sdk.WrapError(err, "cannot delete user")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

// GetUserHandler returns a specific user's information
func (api *API) getUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsername"]

		consumer := getAPIConsumer(ctx)

		var u *sdk.AuthentifiedUser
		var err error
		if username == "me" {
			u, err = user.LoadByID(ctx, api.mustDB(), consumer.AuthentifiedUserID, user.LoadOptions.Default)
		} else {
			u, err = user.LoadByUsername(ctx, api.mustDB(), username, user.LoadOptions.Default)
		}
		if err != nil {
			return err
		}

		return service.WriteJSON(w, u, http.StatusOK)
	}
}

// UpdateUserHandler modifies user informations
func (api *API) updateUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//vars := mux.Vars(r)
		//username := vars["username"]
		//
		//if !deprecatedGetUser(ctx).Admin && username != deprecatedGetUser(ctx).Username {
		//	return service.WriteJSON(w, nil, http.StatusForbidden)
		//}
		//
		//usr, err := user.LoadByUsername(api.mustDB(), username)
		//if err != nil {
		//	return sdk.WrapError(err, "repositoriesManagerAuthorizeCallback> Cannot load user %s", username)
		//}
		//
		//u, err := user.GetDeprecatedUser(api.mustDB(), usr)
		//if err != nil {
		//	return err
		//}
		//
		//var userBody sdk.User
		//if err := service.UnmarshalBody(r, &userBody); err != nil {
		//	return err
		//}
		//
		//userBody.ID = userDB.ID
		//
		//if !user.IsValidEmail(userBody.Email) {
		//	return sdk.WrapError(sdk.ErrWrongRequest, "updateUserHandler: Email address %s is not valid", userBody.Email)
		//}
		//
		//if err := user.UpdateUser(api.mustDB(), userBody); err != nil {
		//	return sdk.WrapError(err, "updateUserHandler: Cannot update user table")
		//}
		//
		//return service.WriteJSON(w, userBody, http.StatusOK)
		return nil
	}
}

// GetUsers fetches all users from databases
func (api *API) getUsersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		users, err := user.LoadAll(ctx, api.mustDB(), user.LoadOptions.WithContacts)
		if err != nil {
			return sdk.WrapError(err, "cannot load user from db")
		}
		return service.WriteJSON(w, users, http.StatusOK)
	}
}
