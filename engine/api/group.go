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

func (api *API) getGroupsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var groups []sdk.Group
		var err error

		withoutDefault := FormBool(r, "withoutDefault")
		if isMaintainer(ctx) {
			groups, err = group.LoadAll(ctx, api.mustDB())
		} else {
			groups, err = group.LoadAllByDeprecatedUserID(ctx, api.mustDB(), getAPIConsumer(ctx).AuthentifiedUser.OldUserStruct.ID)
		}
		if err != nil {
			return err
		}

		// withoutDefault is use by project add, to avoid selecting the default group on project creation
		if withoutDefault {
			var filteredGroups []sdk.Group
			for _, g := range groups {
				if !group.IsDefaultGroupID(g.ID) {
					filteredGroups = append(filteredGroups, g)
				}
			}
			return service.WriteJSON(w, filteredGroups, http.StatusOK)
		}

		return service.WriteJSON(w, groups, http.StatusOK)
	}
}

func (api *API) getGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["permGroupName"]

		g, err := group.LoadByName(ctx, api.mustDB(), name, group.LoadOptions.Default)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, g, http.StatusOK)
	}
}

func (api *API) postGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var data sdk.Group
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot begin tx")
		}
		defer tx.Rollback()

		existingGroup, err := group.LoadByName(ctx, tx, data.Name)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if existingGroup != nil {
			return sdk.WithStack(sdk.ErrGroupPresent)
		}

		consumer := getAPIConsumer(ctx)
		newGroup, err := group.Create(tx, data, consumer.AuthentifiedUser.OldUserStruct.ID)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit tx")
		}

		if err := group.LoadOptions.Default(ctx, api.mustDB(), newGroup); err != nil {
			return err
		}

		return service.WriteJSON(w, newGroup, http.StatusCreated)
	}
}

func (api *API) putGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		groupName := vars["permGroupName"]

		var data sdk.Group
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback()

		oldGroup, err := group.LoadByName(ctx, tx, groupName)
		if err != nil {
			return sdk.WrapError(err, "cannot load group: %s", groupName)
		}

		// In case of rename, checks that new name is not already used
		if data.Name != oldGroup.Name {
			exstingGroup, err := group.LoadByName(ctx, tx, data.Name)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}
			if exstingGroup != nil {
				return sdk.WithStack(sdk.ErrGroupPresent)
			}
		}

		newGroup := *oldGroup
		newGroup.Name = data.Name

		if err := group.Update(tx, &newGroup); err != nil {
			return sdk.WrapError(err, "cannot update group with id: %d", newGroup.ID)
		}

		// TODO Update all requirements that was using the group name

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		// Load extra data for group
		if err := group.LoadOptions.Default(ctx, api.mustDB(), &newGroup); err != nil {
			return err
		}

		return service.WriteJSON(w, newGroup, http.StatusOK)
	}
}

func (api *API) deleteGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["permGroupName"]
		u := getAPIConsumer(ctx)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback()

		g, err := group.LoadByName(ctx, tx, name)
		if err != nil {
			return sdk.WrapError(err, "cannot load %s", name)
		}

		projPerms, err := project.LoadPermissions(tx, g.ID)
		if err != nil {
			return sdk.WrapError(err, "cannot load projects for group")
		}

		if err := group.Delete(ctx, tx, g); err != nil {
			return sdk.WrapError(err, "cannot delete group")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		// TODO refact events
		groupPerm := sdk.GroupPermission{Group: *g}
		for _, pg := range projPerms {
			event.PublishDeleteProjectPermission(&pg.Project, groupPerm, u)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) postGroupUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		groupName := vars["permGroupName"]

		var data sdk.AuthentifiedUser
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if data.ID == "" && data.Username == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given user id or username")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()

		g, err := group.LoadByName(ctx, tx, groupName)
		if err != nil {
			return sdk.WrapError(err, "cannot load group with name: %s", groupName)
		}

		var u *sdk.AuthentifiedUser
		if data.ID != "" {
			u, err = user.LoadByID(ctx, tx, data.ID, user.LoadOptions.WithDeprecatedUser)
		} else {
			u, err = user.LoadByUsername(ctx, tx, data.Username, user.LoadOptions.WithDeprecatedUser)
		}
		if err != nil {
			return err
		}

		// If the user is already in group return an error
		link, err := group.LoadLinkGroupUserForGroupIDAndUserID(ctx, tx, g.ID, u.OldUserStruct.ID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if link != nil {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "given user is already in group")
		}

		// Create the link between group and user with admin flag from request
		if err := group.InsertLinkGroupUser(tx, &group.LinkGroupUser{
			GroupID: g.ID,
			UserID:  u.OldUserStruct.ID,
			Admin:   data.GroupAdmin,
		}); err != nil {
			return sdk.WrapError(err, "cannot add user %s in group %s", u.Username, g.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		// Load extra data for group
		if err := group.LoadOptions.Default(ctx, api.mustDB(), g); err != nil {
			return err
		}

		return service.WriteJSON(w, g, http.StatusCreated)
	}
}

func (api *API) putGroupUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		groupName := vars["permGroupName"]
		username := vars["username"]

		var data sdk.AuthentifiedUser
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()

		g, err := group.LoadByName(ctx, tx, groupName)
		if err != nil {
			return sdk.WrapError(err, "cannot load group with name: %s", groupName)
		}

		u, err := user.LoadByUsername(ctx, tx, username, user.LoadOptions.WithDeprecatedUser)
		if err != nil {
			return err
		}

		link, err := group.LoadLinkGroupUserForGroupIDAndUserID(ctx, tx, g.ID, u.OldUserStruct.ID)
		if err != nil {
			return err
		}

		// In case we are removing admin rights to user, we need to check that it's not the last admin
		if link.Admin && !data.GroupAdmin {
			links, err := group.LoadLinksGroupUserForGroupIDs(ctx, tx, []int64{g.ID})
			if err != nil {
				return err
			}

			var adminFound bool
			for i := range links {
				if links[i].UserID != u.OldUserStruct.ID && links[i].Admin {
					adminFound = true
					break
				}
			}
			if !adminFound {
				return sdk.NewErrorFrom(sdk.ErrGroupNeedAdmin, "cannot remove the last admin of the group")
			}
		}

		link.Admin = data.GroupAdmin

		if err := group.UpdateLinkGroupUser(tx, link); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		// Load extra data for group
		if err := group.LoadOptions.Default(ctx, api.mustDB(), g); err != nil {
			return err
		}

		return service.WriteJSON(w, g, http.StatusOK)
	}
}

func (api *API) deleteGroupUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		groupName := vars["permGroupName"]
		username := vars["username"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()

		g, err := group.LoadByName(ctx, tx, groupName)
		if err != nil {
			return sdk.WrapError(err, "cannot load group with name: %s", groupName)
		}

		u, err := user.LoadByUsername(ctx, tx, username, user.LoadOptions.WithDeprecatedUser)
		if err != nil {
			return err
		}

		link, err := group.LoadLinkGroupUserForGroupIDAndUserID(ctx, tx, g.ID, u.OldUserStruct.ID)
		if err != nil {
			return err
		}

		// In case we are removing an admin from the group, we need to check that it's not the last admin
		if link.Admin {
			links, err := group.LoadLinksGroupUserForGroupIDs(ctx, tx, []int64{g.ID})
			if err != nil {
				return err
			}

			var adminFound bool
			for i := range links {
				if links[i].UserID != u.OldUserStruct.ID && links[i].Admin {
					adminFound = true
					break
				}
			}
			if !adminFound {
				return sdk.NewErrorFrom(sdk.ErrGroupNeedAdmin, "cannot remove the last admin of the group")
			}
		}

		if err := group.DeleteLinkGroupUser(tx, link); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		// In case where the user remove himself from group, do not return it
		if link.UserID == getAPIConsumer(ctx).AuthentifiedUser.OldUserStruct.ID {
			return service.WriteJSON(w, nil, http.StatusOK)
		}

		// Load extra data for group
		if err := group.LoadOptions.Default(ctx, api.mustDB(), g); err != nil {
			return err
		}

		return service.WriteJSON(w, g, http.StatusOK)
	}
}
