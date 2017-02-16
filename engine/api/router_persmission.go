package main

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

// loadUserPermissions retrieves all group memberships
func loadUserPermissions(db gorp.SqlExecutor, user *sdk.User) error {
	user.Groups = nil
	query := `SELECT "group".id, "group".name, "group_user".group_admin FROM "group"
	 		  JOIN group_user ON group_user.group_id = "group".id
	 		  WHERE group_user.user_id = $1 ORDER BY "group".name ASC`

	rows, err := db.Query(query, user.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var group sdk.Group
		var admin bool
		err = rows.Scan(&group.ID, &group.Name, &admin)
		if err != nil {
			return err
		}

		err = project.LoadProjectByGroup(db, &group)
		if err != nil {
			return err
		}

		err = pipeline.LoadPipelineByGroup(db, &group)
		if err != nil {
			return err
		}

		err = application.LoadApplicationByGroup(db, &group)
		if err != nil {
			return err
		}

		err = environment.LoadEnvironmentByGroup(db, &group)
		if err != nil {
			return err
		}

		if admin {
			usr := *user
			usr.Groups = nil
			group.Admins = append(group.Admins, usr)
		}

		user.Groups = append(user.Groups, group)
	}
	return nil
}

// loadGroupPermissions retrieves all group memberships
func loadGroupPermissions(db gorp.SqlExecutor, groupID int64) (*sdk.Group, error) {
	query := `SELECT "group".name FROM "group" WHERE "group".id = $1`

	group := &sdk.Group{ID: groupID}
	err := db.QueryRow(query, groupID).Scan(&group.Name)
	if err != nil {
		return nil, fmt.Errorf("no group with id %d: %s", groupID, err)
	}

	err = project.LoadProjectByGroup(db, group)
	if err != nil {
		return nil, err
	}

	err = pipeline.LoadPipelineByGroup(db, group)
	if err != nil {
		return nil, err
	}

	err = application.LoadApplicationByGroup(db, group)
	if err != nil {
		return nil, err
	}

	err = environment.LoadEnvironmentByGroup(db, group)
	if err != nil {
		return nil, err
	}

	return group, nil
}
