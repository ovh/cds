package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get group name in URL
		vars := mux.Vars(r)
		name := vars["permGroupName"]

		g, errl := group.LoadGroup(api.mustDB(), name)
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load group from db")
		}

		if err := group.LoadUserGroup(api.mustDB(), g); err != nil {
			return sdk.WrapError(err, "Cannot load user group from db")
		}

		isGroupAdmin := false
		currentUser := deprecatedGetUser(ctx)
		for _, grAdmin := range g.Admins {
			if currentUser.ID == grAdmin.ID {
				isGroupAdmin = true
			}
		}
		if isGroupAdmin {
			tokens, errT := group.LoadTokens(api.mustDB(), name)
			if errT != nil {
				return sdk.WrapError(errT, "Cannot load tokens group from db")
			}
			g.Tokens = tokens
		}

		return service.WriteJSON(w, g, http.StatusOK)
	}
}

func (api *API) deleteGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get group name in URL
		vars := mux.Vars(r)
		name := vars["permGroupName"]
		u := deprecatedGetUser(ctx)

		g, errl := group.LoadGroup(api.mustDB(), name)
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load %s", name)
		}

		projPerms, err := project.LoadPermissions(api.mustDB(), g.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot load projects for group")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.DeleteGroupAndDependencies(tx, g); err != nil {
			return sdk.WrapError(err, "Cannot delete group")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		groupPerm := sdk.GroupPermission{Group: *g}
		for _, pg := range projPerms {
			event.PublishDeleteProjectPermission(&pg.Project, groupPerm, u)
		}

		return nil
	}
}

func (api *API) updateGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get group name in URL
		vars := mux.Vars(r)
		oldName := vars["permGroupName"]

		var updatedGroup sdk.Group
		if err := service.UnmarshalBody(r, &updatedGroup); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal")
		}

		if len(updatedGroup.Admins) == 0 {
			return sdk.WrapError(sdk.ErrGroupNeedAdmin, "Cannot Delete all admins for group %s", updatedGroup.Name)
		}

		g, errl := group.LoadGroup(api.mustDB(), oldName)
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load %s", oldName)
		}

		updatedGroup.ID = g.ID
		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.UpdateGroup(tx, &updatedGroup, oldName); err != nil {
			return sdk.WrapError(err, "Cannot update group %s", oldName)
		}

		if err := group.DeleteGroupUserByGroup(tx, &updatedGroup); err != nil {
			return sdk.WrapError(err, "Cannot delete users in group %s", oldName)
		}

		for _, a := range updatedGroup.Admins {
			u, errlu := user.LoadUserWithoutAuth(tx, a.Username)
			if errlu != nil {
				return sdk.WrapError(errlu, "Cannot load user(admins) %s", a.Username)
			}

			if err := group.InsertUserInGroup(tx, updatedGroup.ID, u.ID, true); err != nil {
				return sdk.WrapError(err, "Cannot insert admin %s in group %s", a.Username, updatedGroup.Name)
			}
		}

		for _, a := range updatedGroup.Users {
			u, errlu := user.LoadUserWithoutAuth(tx, a.Username)
			if errlu != nil {
				return sdk.WrapError(errlu, "Cannot load user(members) %s", a.Username)
			}

			if err := group.InsertUserInGroup(tx, updatedGroup.ID, u.ID, false); err != nil {
				return sdk.WrapError(err, "Cannot insert member %s in group %s", a.Username, updatedGroup.Name)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.WriteJSON(w, updatedGroup, http.StatusOK)
	}
}

func (api *API) getGroupsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var groups []sdk.Group
		var err error

		public := FormBool(r, "withPublic")
		withoutDefault := FormBool(r, "withoutDefault")
		if deprecatedGetUser(ctx).Admin {
			groups, err = group.LoadGroups(api.mustDB())
		} else {
			groups, err = group.LoadGroupByUser(api.mustDB(), deprecatedGetUser(ctx).ID)
			if public {
				publicGroups, errl := group.LoadPublicGroups(api.mustDB())
				if errl != nil {
					return sdk.WrapError(errl, "Cannot load group from db")
				}
				groups = append(groups, publicGroups...)
			}
		}
		if err != nil {
			return sdk.WrapError(err, "Cannot load group from db")
		}

		// withoutDefault is use by project Add, to avoid
		// user select the default group on project creation
		if withoutDefault {
			var filteredGroups []sdk.Group
			for _, g := range groups {
				if group.IsDefaultGroupID(g.ID) {
					continue
				} else {
					filteredGroups = append(filteredGroups, g)
				}
			}
			return service.WriteJSON(w, filteredGroups, http.StatusOK)
		}

		return service.WriteJSON(w, groups, http.StatusOK)
	}
}

func (api *API) getPublicGroupsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		groups, err := group.LoadPublicGroups(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "Cannot load group from db")
		}
		return service.WriteJSON(w, groups, http.StatusOK)
	}
}

func (api *API) addGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		g := &sdk.Group{}
		if err := service.UnmarshalBody(r, g); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "Cannot begin tx")
		}
		defer tx.Rollback()

		if _, _, err := group.AddGroup(tx, g); err != nil {
			return sdk.WrapError(err, "Cannot add group")
		}

		// Add caller into group
		if err := group.InsertUserInGroup(tx, g.ID, deprecatedGetUser(ctx).ID, false); err != nil {
			return sdk.WrapError(err, "Cannot add user %s in group %s", deprecatedGetUser(ctx).Username, g.Name)
		}
		// and set it admin
		if err := group.SetUserGroupAdmin(tx, g.ID, deprecatedGetUser(ctx).ID); err != nil {
			return sdk.WrapError(err, "Cannot set user group admin")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit tx")
		}

		return service.WriteJSON(w, g, http.StatusCreated)
	}
}

func (api *API) removeUserFromGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get group name in URL
		vars := mux.Vars(r)
		name := vars["permGroupName"]
		userName := vars["user"]

		g, errl := group.LoadGroup(api.mustDB(), name)
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load %s", name)
		}

		userID, err := user.FindUserIDByName(api.mustDB(), userName)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Unknown user %s", userName)
		}

		userInGroup, errc := group.CheckUserInGroup(api.mustDB(), g.ID, userID)
		if errc != nil {
			return sdk.WrapError(errc, "Cannot check if user %s is already in the group %s", userName, g.Name)
		}

		if !userInGroup {
			return sdk.WrapError(sdk.ErrWrongRequest, "User %s is not in group %s", userName, name)
		}

		if err := group.DeleteUserFromGroup(api.mustDB(), g.ID, userID); err != nil {
			return sdk.WrapError(err, "Cannot delete user %s from group %s", userName, g.Name)
		}

		return nil
	}
}

func (api *API) addUserInGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get group name in URL
		vars := mux.Vars(r)
		name := vars["permGroupName"]

		var users []string
		if err := service.UnmarshalBody(r, &users); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal")
		}

		g, errl := group.LoadGroup(api.mustDB(), name)
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load group %s", name)
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return errb
		}
		defer tx.Rollback()

		for _, u := range users {
			userID, errf := user.FindUserIDByName(api.mustDB(), u)
			if errf != nil {
				return sdk.WrapError(errf, "Unknown user '%s'", u)
			}
			userInGroup, errc := group.CheckUserInGroup(api.mustDB(), g.ID, userID)
			if errc != nil {
				return sdk.WrapError(errc, "Cannot check if user %s is already in the group %s", u, g.Name)
			}
			if !userInGroup {
				if err := group.InsertUserInGroup(api.mustDB(), g.ID, userID, false); err != nil {
					return sdk.WrapError(err, "Cannot add user %s in group %s", u, g.Name)
				}
			}
		}

		return tx.Commit()
	}
}

func (api *API) setUserGroupAdminHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get group name in URL
		vars := mux.Vars(r)
		name := vars["permGroupName"]
		userName := vars["user"]

		g, errl := group.LoadGroup(api.mustDB(), name)
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load %s", name)
		}

		userID, errf := user.FindUserIDByName(api.mustDB(), userName)
		if errf != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Unknown user %s: %s", userName, errf)
		}

		if err := group.SetUserGroupAdmin(api.mustDB(), g.ID, userID); err != nil {
			return sdk.WrapError(err, "Cannot set user group admin")
		}

		return nil
	}
}

func (api *API) removeUserGroupAdminHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get group name in URL
		vars := mux.Vars(r)
		name := vars["permGroupName"]
		userName := vars["user"]

		g, errl := group.LoadGroup(api.mustDB(), name)
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load group %s", name)
		}

		userID, errf := user.FindUserIDByName(api.mustDB(), userName)
		if errf != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Unknown user %s: %s", userName, errf)
		}

		if err := group.RemoveUserGroupAdmin(api.mustDB(), g.ID, userID); err != nil {
			return sdk.WrapError(err, "Cannot remove user group admin privilege")
		}

		return nil
	}
}
