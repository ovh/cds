package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/link"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// getAdminUsersHandler searches CDS users by external link criteria, available for admin only.
func (api *API) getAdminUsersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		consumerType := r.FormValue("consumerType")
		externalUsername := r.FormValue("externalUsername")
		if consumerType == "" || externalUsername == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "consumerType and externalUsername are required")
		}
		if !sdk.AuthConsumerType(consumerType).IsValid() {
			return sdk.WithStack(sdk.ErrInvalidData)
		}

		links, err := link.LoadUserLinksByTypeAndUsername(ctx, api.mustDB(), consumerType, externalUsername)
		if err != nil {
			return err
		}

		userIDs := make([]string, 0, len(links))
		seen := make(map[string]struct{}, len(links))
		for _, l := range links {
			if _, ok := seen[l.AuthentifiedUserID]; ok {
				continue
			}
			seen[l.AuthentifiedUserID] = struct{}{}
			userIDs = append(userIDs, l.AuthentifiedUserID)
		}

		users, err := user.LoadAllByIDs(ctx, api.mustDB(), userIDs)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, users, http.StatusOK)
	}
}

// postUserHandler creates a users, available from admin cdsctl only.
func (api *API) postUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var reqUser sdk.CreateUser
		if err := service.UnmarshalBody(r, &reqUser); err != nil {
			return err
		}

		trackSudo(ctx, w)

		if reqUser.Fullname == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing fullname")
		}
		if reqUser.Username == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid username")
		}
		if reqUser.Email == "" || !sdk.IsValidEmail(reqUser.Email) {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid email")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Check that user don't already exists in database
		existingUser, err := user.LoadByUsername(ctx, tx, reqUser.Username)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrUserNotFound) {
			return err
		}
		if existingUser != nil {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot create a user with given username")
		}

		// Check that user contact don't already exists in database for given email
		existingEmail, err := user.LoadContactByTypeAndValue(ctx, tx, sdk.UserContactTypeEmail, reqUser.Email)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if existingEmail != nil {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot create a user with given email")
		}

		// check organization
		orgAllowed := api.Config.Auth.AllowedOrganizations.Contains(reqUser.Organization)
		if !orgAllowed {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "user organization %q is not allowed", reqUser.Organization)
		}

		// Prepare new user
		newUser := sdk.AuthentifiedUser{
			Ring:     sdk.UserRingUser,
			Username: reqUser.Username,
			Fullname: reqUser.Fullname,
		}

		// Insert the new user in database
		if err := user.Insert(ctx, tx, &newUser); err != nil {
			return err
		}

		userContact := sdk.UserContact{
			Primary:  true,
			Type:     sdk.UserContactTypeEmail,
			UserID:   newUser.ID,
			Value:    reqUser.Email,
			Verified: true,
		}

		// Insert the primary contact for the new user in database
		if err := user.InsertContact(ctx, tx, &userContact); err != nil {
			return err
		}

		if err := api.userSetOrganization(ctx, tx, &newUser, reqUser.Organization); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		newUser.Organization = reqUser.Organization

		return service.WriteJSON(w, newUser, http.StatusCreated)
	}
}

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

		consumer := getUserConsumer(ctx)

		var u *sdk.AuthentifiedUser
		var err error
		if username == "me" {
			u, err = user.LoadByID(ctx, api.mustDB(), consumer.AuthConsumerUser.AuthentifiedUserID, user.LoadOptions.WithOrganization)
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

		consumer := getUserConsumer(ctx)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		var oldUser *sdk.AuthentifiedUser
		if username == "me" {
			oldUser, err = user.LoadByID(ctx, tx, consumer.AuthConsumerUser.AuthentifiedUserID)
		} else {
			oldUser, err = user.LoadByUsername(ctx, tx, username)
		}
		if err != nil {
			return err
		}

		if oldUser.Ring == sdk.UserRingAdmin {
			// Specific audit log for admin: don't change it
			log.Info(ctx, "Administrator has been updated (id=%s) (username: %q -> %q, fullname: %q -> %q, ring: %q -> %q, organization: %q -> %q)",
				oldUser.ID,
				oldUser.Username, data.Username,
				oldUser.Fullname, data.Fullname,
				oldUser.Ring, data.Ring,
				oldUser.Organization, data.Organization,
			)
		}

		newUser := *oldUser

		if oldUser.Username != data.Username {
			// Only an admin can change the username
			if isAdmin(ctx) {
				trackSudo(ctx, w)
				log.Info(ctx, "putUserHandler> %s change username of user %s from %s to %s", consumer.AuthConsumerUser.AuthentifiedUserID, oldUser.ID, oldUser.Username, data.Username)
				newUser.Username = data.Username
			} else {
				return sdk.WithStack(sdk.ErrForbidden)
			}
		}

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
				// An admin can't be disabled, so a disabled user can't become an admin
				if oldUser.Disabled && data.Disabled {
					return sdk.NewErrorFrom(sdk.ErrForbidden, "can't set the admin ring on a disabled user, enable it first")
				}
				if err := authentication.ConsumerRestoreInvalidatedGroupsForUser(ctx, tx, oldUser.ID); err != nil {
					return err
				}
			}

			newUser.Ring = data.Ring
			// Specific audit log for admin: don't change it
			log.Info(ctx, "putUserHandler> %s change ring of user %s from %s to %s", consumer.AuthConsumerUser.AuthentifiedUserID, oldUser.ID, oldUser.Ring, newUser.Ring)
		}

		// Only an admin can disable or enable a user
		if isAdmin(ctx) && oldUser.Disabled != data.Disabled {
			trackSudo(ctx, w)
			if data.Disabled {
				// newUser already holds the ring change of the current request, if any
				if err := api.checkUserCanBeDisabled(ctx, tx, &newUser); err != nil {
					return err
				}
			}
			newUser.Disabled = data.Disabled
			// Specific audit log for admin: don't change it
			log.Info(ctx, "putUserHandler> %s change disabled of user %s from %t to %t", consumer.AuthConsumerUser.AuthentifiedUserID, oldUser.ID, oldUser.Disabled, newUser.Disabled)
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

		// Revoke all sessions of a user that has just been disabled, so that it is
		// signed out right away instead of waiting for its sessions to expire.
		if newUser.Disabled && !oldUser.Disabled {
			if err := revokeUserSessions(ctx, tx, newUser.ID); err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event_v2.PublishUserEvent(ctx, api.Cache, sdk.EventUserUpdated, newUser)

		if err := user.LoadOptions.WithOrganization(ctx, api.mustDBWithCtx(ctx), &newUser); err != nil {
			return err
		}

		return service.WriteJSON(w, newUser, http.StatusOK)
	}
}

// checkUserCanBeDisabled returns an error if disabling the given user would lock the
// instance out: an admin losing its access, or the running services losing the user
// their consumers are attached to.
func (api *API) checkUserCanBeDisabled(ctx context.Context, db gorp.SqlExecutor, u *sdk.AuthentifiedUser) error {
	// An admin can never be disabled, it has to be demoted first. As only an admin can
	// disable a user, this also prevents an admin from locking itself out.
	if u.Ring == sdk.UserRingAdmin {
		return sdk.NewErrorFrom(sdk.ErrForbidden, "can't disable an admin, change its ring first")
	}

	// The consumers of the CDS services are attached to a user, disabling it would
	// prevent every one of them from signing in again.
	consumers, err := authentication.LoadUserConsumersByUserID(ctx, db, u.ID)
	if err != nil {
		return err
	}
	var serviceNames []string
	for i := range consumers {
		if consumers[i].AuthConsumerUser.ServiceName != nil {
			serviceNames = append(serviceNames, *consumers[i].AuthConsumerUser.ServiceName)
		}
	}
	if len(serviceNames) > 0 {
		return sdk.NewErrorFrom(sdk.ErrForbidden, "can't disable user %s, it holds the consumers of services: %s", u.Username, strings.Join(serviceNames, ", "))
	}

	// A disabled user keeps its group memberships, so it can remain the last admin of a
	// group. This is not blocking, unlike a user deletion, but worth an audit log.
	gus, err := group.LoadLinksGroupUserForUserIDs(ctx, db, []string{u.ID})
	if err != nil {
		return err
	}
	var adminGroupIDs []int64
	for i := range gus {
		if gus[i].Admin {
			adminGroupIDs = append(adminGroupIDs, gus[i].GroupID)
		}
	}
	if len(adminGroupIDs) > 0 {
		log.Warn(ctx, "checkUserCanBeDisabled> user %s is disabled but remains group admin of groups %v", u.Username, adminGroupIDs)
	}

	return nil
}

// revokeUserSessions removes every session of every consumer of the given user.
func revokeUserSessions(ctx context.Context, db gorpmapper.SqlExecutorWithTx, userID string) error {
	consumers, err := authentication.LoadUserConsumersByUserID(ctx, db, userID)
	if err != nil {
		return err
	}
	sessions, err := authentication.LoadSessionsByConsumerIDs(ctx, db, sdk.AuthConsumersToIDs(consumers))
	if err != nil {
		return err
	}
	for _, s := range sessions {
		if err := authentication.DeleteSessionByID(db, s.ID); err != nil {
			return err
		}
	}
	return nil
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

		consumer := getUserConsumer(ctx)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		var u *sdk.AuthentifiedUser
		if username == "me" {
			u, err = user.LoadByID(ctx, tx, consumer.AuthConsumerUser.AuthentifiedUserID)
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
				adminGroupIDs = append(adminGroupIDs, gus[i].GroupID)
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
			var lastAdminGroupIDs []int64
			for _, id := range adminGroupIDs {
				if adminLeftCount[id] < 1 {
					lastAdminGroupIDs = append(lastAdminGroupIDs, id)
				}
			}
			if len(lastAdminGroupIDs) > 0 {
				gs, err := group.LoadAllByIDs(ctx, tx, lastAdminGroupIDs)
				if err != nil {
					return err
				}
				mGroups := gs.ToMap()
				names := make([]string, 0, len(lastAdminGroupIDs))
				for _, id := range lastAdminGroupIDs {
					names = append(names, mGroups[id].Name)
				}
				return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot remove user because it is the last admin of group(s): %s", strings.Join(names, ", "))
			}
		}

		if err := user.DeleteByID(tx, u.ID); err != nil {
			return sdk.WrapError(err, "cannot delete user")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event_v2.PublishUserEvent(ctx, api.Cache, sdk.EventUserDeleted, *u)

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}
