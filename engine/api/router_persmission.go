package api

import (
	"fmt"
	"strconv"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func loadGroupPermissionInUser(db gorp.SqlExecutor, groupID int64, u *sdk.User) error {
	permProj, err := project.LoadPermissions(db, groupID)
	if err != nil {
		return sdk.WrapError(err, "loadUserPermissions> Unable to load project permissions for %s", u.Username)
	}
	if u.Permissions.ProjectsPerm == nil {
		u.Permissions.ProjectsPerm = make(map[string]int, len(permProj))
	}
	for _, p := range permProj {
		u.Permissions.ProjectsPerm[p.Project.Key] = p.Permission
	}

	permPip, err := pipeline.LoadPipelineByGroup(db, groupID)
	if err != nil {
		return sdk.WrapError(err, "loadUserPermissions> Unable to load pipeline permissions for %s", u.Username)
	}
	if u.Permissions.PipelinesPerm == nil {
		u.Permissions.PipelinesPerm = make(map[sdk.UserPermissionKey]int, len(permPip))
	}
	for _, p := range permPip {
		u.Permissions.PipelinesPerm[sdk.UserPermissionKey{Key: p.Pipeline.ProjectKey, Name: p.Pipeline.Name}] = p.Permission
	}

	permApp, err := application.LoadPermissions(db, groupID)
	if err != nil {
		return sdk.WrapError(err, "loadUserPermissions> Unable to load application permissions for  %s", u.Username)
	}
	if u.Permissions.ApplicationsPerm == nil {
		u.Permissions.ApplicationsPerm = make(map[sdk.UserPermissionKey]int, len(permApp))
	}
	for _, p := range permApp {
		u.Permissions.ApplicationsPerm[sdk.UserPermissionKey{Key: p.Application.ProjectKey, Name: p.Application.Name}] = p.Permission
	}

	permEnv, err := environment.LoadEnvironmentByGroup(db, groupID)
	if err != nil {
		return sdk.WrapError(err, "loadUserPermissions> Unable to load environment permissions for  %s", u.Username)
	}
	if u.Permissions.EnvironmentsPerm == nil {
		u.Permissions.EnvironmentsPerm = make(map[sdk.UserPermissionKey]int, len(permEnv))
	}
	for _, p := range permEnv {
		u.Permissions.EnvironmentsPerm[sdk.UserPermissionKey{Key: p.Environment.ProjectKey, Name: p.Environment.Name}] = p.Permission
	}

	permWorkflow, err := workflow.LoadWorkflowByGroup(db, groupID)
	if err != nil {
		return sdk.WrapError(err, "loadUserPermissions> Unable to load workflow permissions for  %s", u.Username)
	}
	if u.Permissions.WorkflowsPerm == nil {
		u.Permissions.WorkflowsPerm = make(map[sdk.UserPermissionKey]int, len(permEnv))
	}
	for _, p := range permWorkflow {
		u.Permissions.WorkflowsPerm[sdk.UserPermissionKey{Key: p.Workflow.ProjectKey, Name: p.Workflow.Name}] = p.Permission
	}

	return nil
}

// loadUserPermissions retrieves all group memberships
func loadUserPermissions(db gorp.SqlExecutor, store cache.Store, u *sdk.User) error {
	u.Groups = nil
	k := cache.Key("users", u.Username, "perms")
	if !store.Get(k, &u.Permissions) {
		query := `
			SELECT "group".id, "group".name, "group_user".group_admin 
			FROM "group"
	 		JOIN group_user ON group_user.group_id = "group".id
	 		WHERE group_user.user_id = $1 ORDER BY "group".name ASC`

		rows, err := db.Query(query, u.ID)
		if err != nil {
			return sdk.WrapError(err, "loadUserPermissions> Unable to load user groups %s", u.Username)
		}
		defer rows.Close()

		for rows.Next() {
			var group sdk.Group
			var admin bool
			if err := rows.Scan(&group.ID, &group.Name, &admin); err != nil {
				return sdk.WrapError(err, "loadUserPermissions> Unable scan groups %s", u.Username)
			}
			u.Permissions.Groups = append(u.Permissions.Groups, group.Name)
			if admin {
				u.Permissions.Groups = append(u.Permissions.GroupsAdmin, group.Name)
				usr := *u
				usr.Groups = nil
				group.Admins = append(group.Admins, usr)
			}
			if err := loadGroupPermissionInUser(db, group.ID, u); err != nil {
				return err
			}
			u.Groups = append(u.Groups, group)
		}
		store.SetWithTTL(k, u.Groups, 30)
	}
	return nil
}

// loadGroupPermissions retrieves all group memberships
func loadPermissionsByGroupID(db gorp.SqlExecutor, store cache.Store, groupID int64) (sdk.Group, sdk.UserPermissions, error) {
	u := sdk.User{}
	g := sdk.Group{}
	kg := cache.Key("groups", strconv.Itoa(int(groupID)))
	ku := cache.Key("groups", strconv.Itoa(int(groupID)), "perms")
	if !store.Get(kg, &g) {
		query := `SELECT "group".name FROM "group" WHERE "group".id = $1`
		if err := db.QueryRow(query, groupID).Scan(&g.Name); err != nil {
			return g, sdk.UserPermissions{}, fmt.Errorf("no group with id %d: %s", groupID, err)
		}
		store.SetWithTTL(kg, g, 30)
	}

	if !store.Get(ku, &u.Permissions) {
		if err := loadGroupPermissionInUser(db, groupID, &u); err != nil {
			return g, sdk.UserPermissions{}, sdk.WrapError(err, "loadPermissionsByGroupID")
		}
		store.SetWithTTL(ku, u.Permissions, 30)
	}

	return g, u.Permissions, nil
}
