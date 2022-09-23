package api

import (
	"context"
	"github.com/ovh/cds/engine/api/organization"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/lib/pq"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// GetUsers fetches all users from databases
func (api *API) getUsersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		users, err := user.LoadAll(ctx, api.mustDB(), user.LoadOptions.WithOrganization)
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
			u, err = user.LoadByID(ctx, api.mustDB(), consumer.AuthentifiedUserID, user.LoadOptions.WithOrganization)
		} else {
			u, err = user.LoadByUsername(ctx, api.mustDB(), username, user.LoadOptions.WithOrganization)
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
		newUser.Fullname = data.Fullname

		// Only an admin can change the ring of a user
		if isAdmin(ctx) && oldUser.Ring != data.Ring {
			trackSudo(ctx, w)
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
			log.Debug(ctx, "putUserHandler> %s change ring of user %s from %s to %s", consumer.AuthentifiedUserID, oldUser.ID, oldUser.Ring, newUser.Ring)
		}

		if err := user.Update(ctx, tx, &newUser); err != nil {
			if e, ok := sdk.Cause(err).(*pq.Error); ok && e.Code == gorpmapper.ViolateUniqueKeyPGCode {
				return sdk.NewErrorWithStack(e, sdk.ErrUsernamePresent)
			}
			return sdk.WrapError(err, "cannot update user")
		}

		if isAdmin(ctx) && data.Organization != "" && oldUser.Organization != data.Organization {
			trackSudo(ctx, w)
			if err := api.userSetOrganization(ctx, tx, &newUser, data.Organization); err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		if err := user.LoadOptions.WithOrganization(ctx, api.mustDBWithCtx(ctx), &newUser); err != nil {
			return err
		}

		return service.WriteJSON(w, newUser, http.StatusOK)
	}
}

func (api *API) userSetOrganization(ctx context.Context, db gorpmapper.SqlExecutorWithTx, u *sdk.AuthentifiedUser, org string) error {
	if org == "" {
		return nil
	}
	isAllowed := api.Config.Auth.AllowedOrganizations.Contains(org)
	if !isAllowed {
		return sdk.NewErrorFrom(sdk.ErrForbidden, "user organization %q is not allowed", org)
	}

	existingOrg, err := organization.LoadOrganizationByName(ctx, db, org)
	if err != nil {
		return err
	}

	if err := user.LoadOptions.WithOrganization(ctx, db, u); err != nil {
		return err
	}
	if u.Organization != "" {
		if u.Organization == org {
			return nil
		}
		return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot change user organization to %q, value already set to %q", org, u.Organization)
	}

	u.Organization = org
	if err := user.InsertUserOrganization(ctx, db, &user.UserOrganization{
		AuthentifiedUserID: u.ID,
		OrganizationID:     existingOrg.ID,
	}); err != nil {
		return err
	}

	gs, err := group.LoadAllByUserID(ctx, db, u.ID)
	if err != nil {
		return err
	}
	for i := range gs {
		if err := group.EnsureOrganization(ctx, db, &gs[i]); err != nil {
			return err
		}
	}

	return nil
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

		// We can't delete a user if it's the last admin in a group
		var adminGroupIDs []int64
		gus, err := group.LoadLinksGroupUserForUserIDs(ctx, tx, []string{u.ID})
		if err != nil {
			return err
		}
		for i := range gus {
			if gus[i].Admin {
				adminGroupIDs = append(adminGroupIDs, gus[i].ID)
			}
		}
		if len(adminGroupIDs) > 0 {
			gus, err := group.LoadLinksGroupUserForGroupIDs(ctx, tx, adminGroupIDs)
			if err != nil {
				return err
			}
			adminLeftCount := make(map[int64]int)
			for _, id := range adminGroupIDs {
				adminLeftCount[id] = 0
			}
			for i := range gus {
				if gus[i].AuthentifiedUserID != u.ID && gus[i].Admin {
					adminLeftCount[gus[i].GroupID] += 1
				}
			}
			for _, count := range adminLeftCount {
				if count < 1 {
					return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot remove user because it is the last admin of a group")
				}
			}
		}

		if err := user.DeleteByID(tx, u.ID); err != nil {
			return sdk.WrapError(err, "cannot delete user")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}
