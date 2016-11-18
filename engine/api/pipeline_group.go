package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func updateGroupRoleOnPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]
	groupName := vars["group"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	var groupPipeline sdk.GroupPermission
	err = json.Unmarshal(data, &groupPipeline)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	if groupName != groupPipeline.Group.Name {
		WriteError(w, r, sdk.ErrGroupNotFound)
		return
	}

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	g, err := group.LoadGroup(db, groupPipeline.Group.Name)
	if err != nil {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot find %s: %s\n", groupPipeline.Group.Name, err)
		WriteError(w, r, sdk.ErrGroupNotFound)
		return
	}

	groupInPipeline, err := group.CheckGroupInPipeline(db, p.ID, g.ID)
	if err != nil {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot check if group %s is already in the pipeline %s: %s\n", g.Name, p.Name, err)
		WriteError(w, r, err)
		return
	}
	if groupInPipeline {
		if groupPipeline.Permission != permission.PermissionReadWriteExecute {
			permissions, err := group.LoadAllPipelineGroupByRole(db, p.ID, permission.PermissionReadWriteExecute)
			if err != nil {
				log.Warning("updateGroupRoleOnPipelineHandler: Cannot load groups for pipeline %s: %s\n", p.Name, err)
				WriteError(w, r, err)
				return
			}
			if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
				log.Warning("updateGroupRoleOnPipelineHandler: Cannot remove write permission for group %s in pipeline %s\n", g.Name, p.Name)
				WriteError(w, r, sdk.ErrGroupNeedWrite)
				return
			}
		}

		tx, err := db.Begin()
		if err != nil {
			log.Warning("updateGroupRoleOnPipelineHandler: Cannot start transaction: %s\n", err)
			WriteError(w, r, err)
			return
		}
		defer tx.Rollback()

		err = group.UpdateGroupRoleInPipeline(tx, p.ID, g.ID, groupPipeline.Permission)
		if err != nil {
			log.Warning("updateGroupRoleOnPipelineHandler: Cannot add group %s in pipeline %s:  %s\n", g.Name, p.Name, err)
			WriteError(w, r, err)
			return
		}

		err = pipeline.UpdatePipelineLastModified(tx, p)
		if err != nil {
			log.Warning("updateGroupRoleOnPipelineHandler: Cannot update pipeline last_modified date: %s\n", err)
			WriteError(w, r, err)
			return
		}

		err = tx.Commit()
		if err != nil {
			log.Warning("updateGroupRoleOnPipelineHandler: Cannot start transaction: %s\n", err)
			WriteError(w, r, err)
			return
		}

	}
	w.WriteHeader(http.StatusOK)
}

func updateGroupsOnPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	var groupsPermission []sdk.GroupPermission
	err = json.Unmarshal(data, &groupsPermission)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	if len(groupsPermission) == 0 {
		log.Warning("updateGroupsOnPipelineHandler: Cannot remove all groups for pipeline %s", pipelineName)
		WriteError(w, r, sdk.ErrGroupNeedWrite)
		return
	}

	found := false
	for _, gp := range groupsPermission {
		if gp.Permission == permission.PermissionReadWriteExecute {
			found = true
			break
		}
	}
	if !found {
		log.Warning("updateGroupsOnPipelineHandler: Need one group with write permission.")
		WriteError(w, r, sdk.ErrGroupNeedWrite)
		return
	}

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("updateGroupsOnPipelineHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupsOnPipelineHandler: Cannot start transaction: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	defer tx.Rollback()

	err = group.DeleteAllGroupFromPipeline(tx, p.ID)
	if err != nil {
		log.Warning("updateGroupsOnPipelineHandler: Cannot delete groups from pipeline %s: %s\n", p.Name, err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	for _, g := range groupsPermission {
		groupData, err := group.LoadGroup(tx, g.Group.Name)
		if err != nil {
			log.Warning("updateGroupsOnPipelineHandler: Cannot load group %s: %s\n", g.Group.Name, err)
			WriteError(w, r, sdk.ErrUnknownError)
			return
		}
		err = group.InsertGroupInPipeline(tx, p.ID, groupData.ID, g.Permission)
		if err != nil {
			log.Warning("updateGroupsOnPipelineHandler: Cannot insert group %s in pipeline %s: %s\n", g.Group.Name, p.Name, err)
			WriteError(w, r, sdk.ErrUnknownError)
			return
		}
	}

	err = pipeline.UpdatePipelineLastModified(tx, p)
	if err != nil {
		log.Warning("updateGroupsOnPipelineHandler: Cannot update pipeline last_modified date: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateGroupsOnPipelineHandler: Cannot commit transaction: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

}

func addGroupInPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	var groupPermission sdk.GroupPermission
	err = json.Unmarshal(data, &groupPermission)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("addGroupInPipeline: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	g, err := group.LoadGroup(db, groupPermission.Group.Name)
	if err != nil {
		log.Warning("addGroupInPipeline: Cannot find %s: %s\n", groupPermission.Group.Name, err)
		WriteError(w, r, err)
		return
	}

	groupInPipeline, err := group.CheckGroupInPipeline(db, p.ID, g.ID)
	if err != nil {
		log.Warning("addGroupInPipeline: Cannot check if group %s is already in the pipeline %s: %s\n", g.Name, p.Name, err)
		WriteError(w, r, err)
		return
	}
	if !groupInPipeline {

		tx, err := db.Begin()
		if err != nil {
			log.Warning("addGroupInPipeline: Cannot start transaction: %s\n", err)
			WriteError(w, r, err)
			return
		}
		defer tx.Rollback()

		err = group.InsertGroupInPipeline(tx, p.ID, g.ID, groupPermission.Permission)
		if err != nil {
			log.Warning("addGroupInPipeline: Cannot add group %s in pipeline %s:  %s\n", g.Name, p.Name, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = pipeline.UpdatePipelineLastModified(tx, p)
		if err != nil {
			log.Warning("addGroupInPipeline: Cannot update pipeline last_modified date: %s\n", err)
			WriteError(w, r, err)
			return
		}

		err = tx.Commit()
		if err != nil {
			log.Warning("addGroupInPipeline: Cannot commit transaction: %s\n", err)
			WriteError(w, r, err)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func deleteGroupFromPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]
	groupName := vars["group"]

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot find %s: %s\n", groupName, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = group.DeleteGroupFromPipeline(tx, p.ID, g.ID)
	if err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot delete group %s from project %s:  %s\n", g.Name, p.Name, err)
		WriteError(w, r, err)
		return
	}

	err = pipeline.UpdatePipelineLastModified(tx, p)
	if err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot update pipeline last_modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
