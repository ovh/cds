package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getGroupHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]

	g, errl := group.LoadGroup(db, name)
	if errl != nil {
		return sdk.WrapError(errl, "getGroupHandler: Cannot load group from db")
	}

	if err := group.LoadUserGroup(db, g); err != nil {
		return sdk.WrapError(err, "getGroupHandler: Cannot load user group from db")
	}

	return WriteJSON(w, r, g, http.StatusOK)
}

func deleteGroupHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]

	g, errl := group.LoadGroup(db, name)
	if errl != nil {
		return sdk.WrapError(errl, "deleteGroupHandler: Cannot load %s", name)
	}

	if err := project.LoadPermissions(db, g); err != nil {
		return sdk.WrapError(err, "deleteGroupHandler: Cannot load projects for group")
	}

	if err := application.LoadPermissions(db, g); err != nil {
		return sdk.WrapError(err, "deleteGroupHandler: Cannot load application for group")
	}

	if err := pipeline.LoadPipelineByGroup(db, g); err != nil {
		return sdk.WrapError(err, "deleteGroupHandler: Cannot load pipeline for group")
	}

	if err := environment.LoadEnvironmentByGroup(db, g); err != nil {
		return sdk.WrapError(err, "deleteGroupHandler: Cannot load environment for group")
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "deleteGroupHandler> cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.DeleteGroupAndDependencies(tx, g); err != nil {
		return sdk.WrapError(err, "deleteGroupHandler> cannot delete group")
	}

	for _, pg := range g.ProjectGroups {
		if err := project.UpdateLastModified(tx, c.User, &pg.Project); err != nil {
			return sdk.WrapError(err, "deleteGroupHandler> Cannot update project last modified date")
		}
	}

	for _, pg := range g.ApplicationGroups {
		if err := application.UpdateLastModified(tx, &pg.Application, c.User); err != nil {
			return sdk.WrapError(err, "deleteGroupHandler> Cannot update application last modified date")
		}
	}

	for _, pg := range g.PipelineGroups {
		p := &sdk.Project{
			Key: pg.Pipeline.ProjectKey,
		}
		if err := pipeline.UpdatePipelineLastModified(tx, p, &pg.Pipeline, c.User); err != nil {
			return sdk.WrapError(err, "deleteGroupHandler> Cannot update pipeline last modified date")
		}
	}

	for _, pg := range g.EnvironmentGroups {
		if err := environment.UpdateLastModified(tx, c.User, &pg.Environment); err != nil {
			return sdk.WrapError(err, "deleteGroupHandler> Cannot update environment last modified date")
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteGroupHandler> cannot commit transaction")
	}
	return nil
}

func updateGroupHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	oldName := vars["permGroupName"]

	var updatedGroup sdk.Group
	if err := UnmarshalBody(r, &updatedGroup); err != nil {
		return sdk.WrapError(err, "updateGroupHandler> cannot unmarshal")
	}

	if len(updatedGroup.Admins) == 0 {
		return sdk.WrapError(sdk.ErrGroupNeedAdmin, "updateGroupHandler: Cannot Delete all admins for group %s", updatedGroup.Name)
	}

	g, errl := group.LoadGroup(db, oldName)
	if errl != nil {
		return sdk.WrapError(errl, "updateGroupHandler: Cannot load %s", oldName)
	}

	updatedGroup.ID = g.ID
	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "updateGroupHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.UpdateGroup(tx, &updatedGroup, oldName); err != nil {
		return sdk.WrapError(err, "updateGroupHandler: Cannot update group %s", oldName)
	}

	if err := group.DeleteGroupUserByGroup(tx, &updatedGroup); err != nil {
		return sdk.WrapError(err, "updateGroupHandler: Cannot delete users in group %s", oldName)
	}

	for _, a := range updatedGroup.Admins {
		u, errlu := user.LoadUserWithoutAuth(tx, a.Username)
		if errlu != nil {
			return sdk.WrapError(errlu, "updateGroupHandler: Cannot load user(admins) %s", a.Username)
		}

		if err := group.InsertUserInGroup(tx, updatedGroup.ID, u.ID, true); err != nil {
			return sdk.WrapError(err, "updateGroupHandler: Cannot insert admin %s in group %s", a.Username, updatedGroup.Name)
		}
	}

	for _, a := range updatedGroup.Users {
		u, errlu := user.LoadUserWithoutAuth(tx, a.Username)
		if errlu != nil {
			return sdk.WrapError(errlu, "updateGroupHandler: Cannot load user(members) %s", a.Username)
		}

		if err := group.InsertUserInGroup(tx, updatedGroup.ID, u.ID, false); err != nil {
			return sdk.WrapError(err, "updateGroupHandler: Cannot insert member %s in group %s", a.Username, updatedGroup.Name)
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateGroupHandler: Cannot commit transaction")
	}

	return nil
}

func getGroups(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	var groups []sdk.Group
	var err error

	public := r.FormValue("withPublic")
	if c.User.Admin {
		groups, err = group.LoadGroups(db)
	} else {
		groups, err = group.LoadGroupByUser(db, c.User.ID)
		if public == "true" {
			publicGroups, errl := group.LoadPublicGroups(db)
			if errl != nil {
				return sdk.WrapError(errl, "GetGroups: Cannot load group from db")
			}
			groups = append(groups, publicGroups...)
		}
	}
	if err != nil {
		return sdk.WrapError(err, "GetGroups: Cannot load group from db")
	}

	return WriteJSON(w, r, groups, http.StatusOK)
}

func getPublicGroups(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	groups, err := group.LoadPublicGroups(db)
	if err != nil {
		return sdk.WrapError(err, "GetGroups: Cannot load group from db")
	}
	return WriteJSON(w, r, groups, http.StatusOK)
}

func addGroupHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	g := &sdk.Group{}
	if err := UnmarshalBody(r, g); err != nil {
		return sdk.WrapError(err, "addGroupHandler> cannot unmarshal")
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "addGroupHandler> cannot begin tx")
	}
	defer tx.Rollback()

	if _, _, err := group.AddGroup(tx, g); err != nil {
		return sdk.WrapError(err, "addGroupHandler> cannot add group")
	}

	// Add caller into group
	if err := group.InsertUserInGroup(tx, g.ID, c.User.ID, false); err != nil {
		return sdk.WrapError(err, "addGroupHandler> cannot add user %s in group %s", c.User.Username, g.Name)
	}
	// and set it admin
	if err := group.SetUserGroupAdmin(tx, g.ID, c.User.ID); err != nil {
		return sdk.WrapError(err, "addGroupHandler> cannot set user group admin")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addGroupHandler> cannot commit tx")
	}

	w.WriteHeader(http.StatusCreated)
	return nil
}

func removeUserFromGroupHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]
	userName := vars["user"]

	g, errl := group.LoadGroup(db, name)
	if errl != nil {
		return sdk.WrapError(errl, "removeUserFromGroupHandler: Cannot load %s", name)
	}

	userID, err := user.FindUserIDByName(db, userName)
	if err != nil {
		return sdk.WrapError(sdk.ErrNotFound, "removeUserFromGroupHandler: Unknown user %s", userName)
	}

	userInGroup, errc := group.CheckUserInGroup(db, g.ID, userID)
	if errc != nil {
		return sdk.WrapError(errc, "removeUserFromGroupHandler: Cannot check if user %s is already in the group %s", userName, g.Name)
	}

	if !userInGroup {
		return sdk.WrapError(sdk.ErrWrongRequest, "User %s is not in group %s", userName, name)
	}

	if err := group.DeleteUserFromGroup(db, g.ID, userID); err != nil {
		return sdk.WrapError(err, "removeUserFromGroupHandler: Cannot delete user %s from group %s", userName, g.Name)
	}

	log.Info("User %s removed from group %s", userName, name)
	return nil
}

func addUserInGroup(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]

	var users []string
	if err := UnmarshalBody(r, &users); err != nil {
		return sdk.WrapError(err, "addGroupHandler> cannot unmarshal")
	}

	g, errl := group.LoadGroup(db, name)
	if errl != nil {
		return sdk.WrapError(errl, "AddUserInGroup: Cannot load %s", name)
	}

	tx, errb := db.Begin()
	if errb != nil {
		return errb
	}
	defer tx.Rollback()

	for _, u := range users {
		userID, errf := user.FindUserIDByName(db, u)
		if errf != nil {
			return sdk.WrapError(errf, "AddUserInGroup: Unknown user '%s'", u)
		}
		userInGroup, errc := group.CheckUserInGroup(db, g.ID, userID)
		if errc != nil {
			return sdk.WrapError(errc, "AddUserInGroup: Cannot check if user %s is already in the group %s", u, g.Name)
		}
		if !userInGroup {
			if err := group.InsertUserInGroup(db, g.ID, userID, false); err != nil {
				return sdk.WrapError(err, "AddUserInGroup: Cannot add user %s in group %s", u, g.Name)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func setUserGroupAdminHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]
	userName := vars["user"]

	g, errl := group.LoadGroup(db, name)
	if errl != nil {
		return sdk.WrapError(errl, "setUserGroupAdminHandler: Cannot load %s", name)
	}

	userID, errf := user.FindUserIDByName(db, userName)
	if errf != nil {
		return sdk.WrapError(sdk.ErrNotFound, "setUserGroupAdminHandler: Unknown user %s: %s", userName, errf)
	}

	if err := group.SetUserGroupAdmin(db, g.ID, userID); err != nil {
		return sdk.WrapError(err, "setUserGroupAdminHandler: cannot set user group admin")
	}

	return nil
}

func removeUserGroupAdminHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]
	userName := vars["user"]

	g, errl := group.LoadGroup(db, name)
	if errl != nil {
		return sdk.WrapError(errl, "removeUserGroupAdminHandler: Cannot load %s", name)
	}

	userID, errf := user.FindUserIDByName(db, userName)
	if errf != nil {
		return sdk.WrapError(sdk.ErrNotFound, "removeUserGroupAdminHandler: Unknown user %s: %s", userName, errf)
	}

	if err := group.RemoveUserGroupAdmin(db, g.ID, userID); err != nil {
		return sdk.WrapError(err, "removeUserGroupAdminHandler: cannot remove user group admin privilege")
	}

	return nil
}
