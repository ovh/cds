package main

import (
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getGroupHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]

	g, err := group.LoadGroup(db, name)
	if err != nil {
		log.Warning("getGroupHandler: Cannot load group from db: %s\n", err)
		return err
	}

	err = group.LoadUserGroup(db, g)
	if err != nil {
		log.Warning("getGroupHandler: Cannot load user group from db: %s\n", err)
		return err
	}

	return WriteJSON(w, r, g, http.StatusOK)
}

func deleteGroupHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]

	g, err := group.LoadGroup(db, name)
	if err != nil {
		log.Warning("deleteGroupHandler: Cannot load %s: %s\n", name, err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteGroupHandler> cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	err = group.DeleteGroupAndDependencies(tx, g)
	if err != nil {
		log.Warning("deleteGroupHandler> cannot delete group: %s\n", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteGroupHandler> cannot commit transaction: %s\n", err)
		return err
	}

	return nil
}

func updateGroupHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	oldName := vars["permGroupName"]

	var updatedGroup sdk.Group
	if err := UnmarshalBody(r, &updatedGroup); err != nil {
		return err
	}

	if len(updatedGroup.Admins) == 0 {
		log.Warning("updateGroupHandler: Cannot Delete all admins for group %s\n", updatedGroup.Name)
		return sdk.ErrGroupNeedAdmin
	}

	g, err := group.LoadGroup(db, oldName)
	if err != nil {
		log.Warning("updateGroupHandler: Cannot load %s: %s\n", oldName, err)
		return err
	}

	updatedGroup.ID = g.ID
	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupHandler: Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	err = group.UpdateGroup(tx, &updatedGroup, oldName)
	if err != nil {
		log.Warning("updateGroupHandler: Cannot update group %s: %s\n", oldName, err)
		return err
	}

	err = group.DeleteGroupUserByGroup(tx, &updatedGroup)
	if err != nil {
		log.Warning("updateGroupHandler: Cannot delete users in group %s: %s\n", oldName, err)
		return err
	}

	for _, a := range updatedGroup.Admins {
		u, err := user.LoadUserWithoutAuth(tx, a.Username)
		if err != nil {
			log.Warning("updateGroupHandler: Cannot load user(admins) %s: %s\n", a.Username, err)
			return err
		}
		err = group.InsertUserInGroup(tx, updatedGroup.ID, u.ID, true)
		if err != nil {
			log.Warning("updateGroupHandler: Cannot insert admin %s in group %s: %s\n", a.Username, updatedGroup.Name, err)
			return err
		}
	}

	for _, a := range updatedGroup.Users {
		u, err := user.LoadUserWithoutAuth(tx, a.Username)
		if err != nil {
			log.Warning("updateGroupHandler: Cannot load user(members) %s: %s\n", a.Username, err)
			return err
		}
		err = group.InsertUserInGroup(tx, updatedGroup.ID, u.ID, false)
		if err != nil {
			log.Warning("updateGroupHandler: Cannot insert member %s in group %s: %s\n", a.Username, updatedGroup.Name, err)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateGroupHandler: Cannot commit transaction: %s\n", err)
		return err
	}

	return nil
}

func getGroups(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	var groups []sdk.Group
	var err error

	public := r.FormValue("withPublic")
	if c.User.Admin {
		groups, err = group.LoadGroups(db)
	} else {
		groups, err = group.LoadGroupByUser(db, c.User.ID)
		if public == "true" {
			publicGroups, errP := group.LoadPublicGroups(db)
			if errP != nil {
				log.Warning("GetGroups: Cannot load group from db: %s\n", errP)
				return errP
			}
			groups = append(groups, publicGroups...)
		}
	}
	if err != nil {
		log.Warning("GetGroups: Cannot load group from db: %s\n", err)
		return err
	}

	return WriteJSON(w, r, groups, http.StatusOK)
}

func getPublicGroups(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	groups, err := group.LoadPublicGroups(db)
	if err != nil {
		log.Warning("GetGroups: Cannot load group from db: %s\n", err)
		return err
	}
	return WriteJSON(w, r, groups, http.StatusOK)
}

func addGroupHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	g := &sdk.Group{}
	if err := UnmarshalBody(r, g); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addGroupHandler> cannot begin tx: %s\n", err)
		return err
	}
	defer tx.Rollback()

	_, _, err = group.AddGroup(tx, g)
	if err != nil {
		log.Warning("addGroupHandler> cannot add group: %s\n", err)
		return err
	}

	// Add caller into group
	err = group.InsertUserInGroup(tx, g.ID, c.User.ID, false)
	if err != nil {
		log.Warning("addGroupHandler> cannot add user %s in group %s: %s\n", c.User.Username, g.Name, err)
		return err
	}
	// and set it admin
	err = group.SetUserGroupAdmin(tx, g.ID, c.User.ID)
	if err != nil {
		log.Warning("addGroupHandler> cannot set user group admin: %s\n", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addGroupHandler> cannot commit tx: %s\n", err)
		return err
	}

	fmt.Printf("POST /group: Group %s added\n", g.Name)
	w.WriteHeader(http.StatusCreated)
	return nil
}

func removeUserFromGroupHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]
	userName := vars["user"]

	g, err := group.LoadGroup(db, name)
	if err != nil {
		log.Warning("removeUserFromGroupHandler: Cannot load %s: %s\n", name, err)
		return err
	}

	userID, err := user.FindUserIDByName(db, userName)
	if err != nil {
		log.Warning("removeUserFromGroupHandler: Unknown user %s: %s\n", userName, err)
		return sdk.ErrNotFound
	}

	userInGroup, err := group.CheckUserInGroup(db, g.ID, userID)
	if err != nil {
		log.Warning("removeUserFromGroupHandler: Cannot check if user %s is already in the group %s: %s\n", userName, g.Name, err)
		return err
	}

	if !userInGroup {
		log.Warning("removeUserFromGroupHandler> User %s is not in group %s\n", userName, name)
		return sdk.ErrWrongRequest
	}

	err = group.DeleteUserFromGroup(db, g.ID, userID)
	if err != nil {
		log.Warning("removeUserFromGroupHandler: Cannot delete user %s from group %s:  %s\n", userName, g.Name, err)
		return err
	}

	log.Notice("User %s removed from group %s\n", userName, name)
	return nil
}

func addUserInGroup(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]

	var users []string
	if err := UnmarshalBody(r, &users); err != nil {
		return err
	}

	g, err := group.LoadGroup(db, name)
	if err != nil {
		log.Warning("AddUserInGroup: Cannot load %s: %s\n", name, err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, u := range users {
		userID, err := user.FindUserIDByName(db, u)
		if err != nil {
			log.Warning("AddUserInGroup: Unknown user '%s': %s\n", u, err)
			return err
		}
		userInGroup, err := group.CheckUserInGroup(db, g.ID, userID)
		if err != nil {
			log.Warning("AddUserInGroup: Cannot check if user %s is already in the group %s: %s\n", u, g.Name, err)
			return err
		}
		if !userInGroup {
			group.InsertUserInGroup(db, g.ID, userID, false)
			if err != nil {
				log.Warning("AddUserInGroup: Cannot add user %s in group %s:  %s\n", u, g.Name, err)
				return err
			}
		}

	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func setUserGroupAdminHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]
	userName := vars["user"]

	g, err := group.LoadGroup(db, name)
	if err != nil {
		log.Warning("setUserGroupAdminHandler: Cannot load %s: %s\n", name, err)
		return err
	}

	userID, err := user.FindUserIDByName(db, userName)
	if err != nil {
		log.Warning("setUserGroupAdminHandler: Unknown user %s: %s\n", userName, err)
		return sdk.ErrNotFound
	}

	err = group.SetUserGroupAdmin(db, g.ID, userID)
	if err != nil {
		log.Warning("setUserGroupAdminHandler: cannot set user group admin: %s\n", err)
		return err
	}

	return nil
}

func removeUserGroupAdminHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get group name in URL
	vars := mux.Vars(r)
	name := vars["permGroupName"]
	userName := vars["user"]

	g, err := group.LoadGroup(db, name)
	if err != nil {
		log.Warning("removeUserGroupAdminHandler: Cannot load %s: %s\n", name, err)
		return err
	}

	userID, err := user.FindUserIDByName(db, userName)
	if err != nil {
		log.Warning("removeUserGroupAdminHandler: Unknown user %s: %s\n", userName, err)
		return sdk.ErrNotFound
	}

	err = group.RemoveUserGroupAdmin(db, g.ID, userID)
	if err != nil {
		log.Warning("removeUserGroupAdminHandler: cannot remove user group admin privilege: %s\n", err)
		return err
	}

	return nil
}
