package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// GetUsers fetches all users from databases
func (api *API) getUsersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		users, err := user.LoadAll(ctx, api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "cannot load user from db")
		}
		return service.WriteJSON(w, users, http.StatusOK)
	}
}

// GetUserHandler returns a specific user's information
func (api *API) getUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsernamePublic"]

		consumer := getAPIConsumer(ctx)

		var u *sdk.AuthentifiedUser
		var err error
		if username == "me" {
			u, err = user.LoadByID(ctx, api.mustDB(), consumer.AuthentifiedUserID)
		} else {
			u, err = user.LoadByUsername(ctx, api.mustDB(), username)
		}
		if err != nil {
			return err
		}

		return service.WriteJSON(w, u, http.StatusOK)
	}
}

func (api *API) putUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsernamePublic"]

		var data sdk.AuthentifiedUser
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		consumer := getAPIConsumer(ctx)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		var oldUser *sdk.AuthentifiedUser
		if username == "me" {
			oldUser, err = user.LoadByID(ctx, tx, consumer.AuthentifiedUserID)
		} else {
			oldUser, err = user.LoadByUsername(ctx, tx, username)
		}
		if err != nil {
			return err
		}

		newUser := *oldUser
		newUser.Username = data.Username
		newUser.Fullname = data.Fullname

		// Only an admin can change the ring of a user
		if isAdmin(ctx) && oldUser.Ring != data.Ring {
			// If previous ring was admin, check that the user is not the last admin
			if oldUser.Ring == sdk.UserRingAdmin {
				count, err := user.CountAdmin(tx)
				if err != nil {
					return err
				}
				if count < 2 {
					return sdk.NewErrorFrom(sdk.ErrForbidden, "can't remove the last admin")
				}

				// Invalidate consumer's group if user is not part of it
				gs, err := group.LoadAllByUserID(ctx, tx, oldUser.ID)
				if err != nil {
					return err
				}
				if err := authentication.ConsumerInvalidateGroupsForUser(ctx, tx, oldUser.ID, gs.ToIDs()); err != nil {
					return err
				}
			}

			// If new ring is admin we need to restore invalid consumer group for user
			if data.Ring == sdk.UserRingAdmin {
				if err := authentication.ConsumerRestoreInvalidatedGroupsForUser(ctx, tx, oldUser.ID); err != nil {
					return err
				}
			}

			newUser.Ring = data.Ring
			log.Debug("putUserHandler> %s change ring of user %s from %s to %s", consumer.AuthentifiedUserID, oldUser.ID, oldUser.Ring, newUser.Ring)
		}

		if err := user.Update(ctx, tx, &newUser); err != nil {
			if e, ok := sdk.Cause(err).(*pq.Error); ok && e.Code == database.ViolateUniqueKeyPGCode {
				return sdk.NewErrorWithStack(e, sdk.ErrUsernamePresent)
			}
			return sdk.WrapError(err, "cannot update user")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		return service.WriteJSON(w, newUser, http.StatusOK)
	}
}

// DeleteUserHandler removes a user.
func (api *API) deleteUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		username := vars["permUsernamePublic"]

		consumer := getAPIConsumer(ctx)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		var u *sdk.AuthentifiedUser
		if username == "me" {
			u, err = user.LoadByID(ctx, tx, consumer.AuthentifiedUserID)
		} else {
			u, err = user.LoadByUsername(ctx, tx, username)
		}
		if err != nil {
			return err
		}

		// We can't delete the last admin
		if u.Ring == sdk.UserRingAdmin {
			count, err := user.CountAdmin(tx)
			if err != nil {
				return err
			}
			if count < 2 {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "can't remove the last admin")
			}
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
