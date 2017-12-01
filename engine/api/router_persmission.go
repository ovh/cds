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

// loadUserPermissions retrieves all group memberships
func loadUserPermissions(db gorp.SqlExecutor, store cache.Store, user *sdk.User) error {
	user.Groups = nil
	k := cache.Key("users", user.Username, "permissions")
	if !store.Get(k, &user.Groups) {
		query := `
			SELECT "group".id, "group".name, "group_user".group_admin 
			FROM "group"
	 		JOIN group_user ON group_user.group_id = "group".id
	 		WHERE group_user.user_id = $1 ORDER BY "group".name ASC`

		rows, err := db.Query(query, user.ID)
		if err != nil {
			return sdk.WrapError(err, "loadUserPermissions> Unable to load user groups %s", user.Username)
		}
		defer rows.Close()

		for rows.Next() {
			var group sdk.Group
			var admin bool
			if err := rows.Scan(&group.ID, &group.Name, &admin); err != nil {
				return sdk.WrapError(err, "loadUserPermissions> Unable scan groups %s", user.Username)
			}
			if err := project.LoadPermissions(db, &group); err != nil {
				return sdk.WrapError(err, "loadUserPermissions> Unable to load project permissions for %s", user.Username)
			}
			if err := pipeline.LoadPipelineByGroup(db, &group); err != nil {
				return sdk.WrapError(err, "loadUserPermissions> Unable to load pipeline permissions for %s", user.Username)
			}
			if err := application.LoadPermissions(db, &group); err != nil {
				return sdk.WrapError(err, "loadUserPermissions> Unable to load application permissions for  %s", user.Username)
			}
			if err := environment.LoadEnvironmentByGroup(db, &group); err != nil {
				return sdk.WrapError(err, "loadUserPermissions> Unable to load environment permissions for  %s", user.Username)
			}
			if err := workflow.LoadWorkflowByGroup(db, &group); err != nil {
				return sdk.WrapError(err, "loadUserPermissions> Unable to load workflow permissions for  %s", user.Username)
			}
			if admin {
				usr := *user
				usr.Groups = nil
				group.Admins = append(group.Admins, usr)
			}
			user.Groups = append(user.Groups, group)
		}
		store.SetWithTTL(k, user.Groups, 30)
	}
	return nil
}

// loadGroupPermissions retrieves all group memberships
func loadGroupPermissions(db gorp.SqlExecutor, store cache.Store, groupID int64) (*sdk.Group, error) {
	group := &sdk.Group{ID: groupID}
	k := cache.Key("groups", strconv.Itoa(int(groupID)), "permissions")
	if !store.Get(k, group) {
		query := `SELECT "group".name FROM "group" WHERE "group".id = $1`
		if err := db.QueryRow(query, groupID).Scan(&group.Name); err != nil {
			return nil, fmt.Errorf("no group with id %d: %s", groupID, err)
		}
		if err := project.LoadPermissions(db, group); err != nil {
			return nil, err
		}
		if err := pipeline.LoadPipelineByGroup(db, group); err != nil {
			return nil, err
		}
		if err := application.LoadPermissions(db, group); err != nil {
			return nil, err
		}
		if err := environment.LoadEnvironmentByGroup(db, group); err != nil {
			return nil, err
		}
		store.SetWithTTL(k, group, 30)
	}
	return group, nil
}
