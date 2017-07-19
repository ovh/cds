package group

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LoadAllEnvironmentGroupByRole load all group for the given environment and role
func LoadAllEnvironmentGroupByRole(db gorp.SqlExecutor, environmentID int64, role int) ([]sdk.GroupPermission, error) {
	groupsPermission := []sdk.GroupPermission{}
	query := `
		SELECT environment_group.group_id, environment_group.role
		FROM environment_group
		JOIN environment ON environment_group.environment_id = environment.id
		WHERE environment.id = $1 AND role = $2;
	`
	rows, err := db.Query(query, environmentID, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var gPermission sdk.GroupPermission
		rows.Scan(&gPermission.Group.ID, &gPermission.Permission)
		groupsPermission = append(groupsPermission, gPermission)
	}
	return groupsPermission, nil
}

// IsInEnvironment checks wether groups already has permissions on environment or not
func IsInEnvironment(db gorp.SqlExecutor, environmentID, groupID int64) (bool, error) {
	query := `SELECT COUNT(id) FROM environment_group
	WHERE environment_id = $1 AND group_id = $2`
	var count int64

	err := db.QueryRow(query, environmentID, groupID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// InsertGroupInEnvironment add permissions on Environment to Group
func InsertGroupInEnvironment(db gorp.SqlExecutor, environmentID, groupID int64, role int) error {
	query := `INSERT INTO environment_group (environment_id, group_id,role) VALUES($1,$2,$3)`
	_, err := db.Exec(query, environmentID, groupID, role)
	return err
}

// InsertGroupsInEnvironment Link the given groups and the given environment
func InsertGroupsInEnvironment(db gorp.SqlExecutor, groupPermission []sdk.GroupPermission, envID int64) error {
	for _, g := range groupPermission {
		if err := InsertGroupInEnvironment(db, envID, g.Group.ID, g.Permission); err != nil {
			log.Warning("InsertGroupsInEnvironment> unable to insert group %d %s on env %d : %s", g.Group.ID, g.Group.Name, envID, err)
			return err
		}
	}
	return nil
}

// UpdateGroupRoleInEnvironment update permission on environment
func UpdateGroupRoleInEnvironment(db gorp.SqlExecutor, key, envName, groupName string, role int) error {
	query := `UPDATE environment_group
	          SET role=$1
	          FROM environment, project, "group"
	          WHERE environment.id = environment_id AND environment.project_id = project.id AND "group".id = group_id
	          AND environment.name = $2 AND  project.projectKey = $3 AND "group".name = $4 `
	if _, err := db.Exec(query, role, envName, key, groupName); err != nil {
		return err
	}

	// Update project
	query = `
		UPDATE project
		SET last_modified = current_timestamp
		WHERE id IN (
			SELECT id
			FROM project
			WHERE projectKey = $1
		)
	`
	_, err := db.Exec(query, key)
	return err
}

// Deprecated
// DeleteAllGroupFromEnvironment remove all group from the given environment
func DeleteAllGroupFromEnvironment(db gorp.SqlExecutor, environmentID int64) error {
	// Update environment
	query := `
		UPDATE environment
		SET last_modified = current_timestamp
		WHERE id = $1
	`
	if _, err := db.Exec(query, environmentID); err != nil {
		return err
	}
	//Delete association
	query = `DELETE FROM environment_group
		  WHERE environment_id=$1`
	_, err := db.Exec(query, environmentID)
	return err
}

// DeleteGroupFromEnvironment removes access to environment to group members
func DeleteGroupFromEnvironment(db gorp.SqlExecutor, key, envName, groupName string) error {
	// Update project
	query := `
		UPDATE project
		SET last_modified = current_timestamp
		WHERE id IN (
			SELECT id
			FROM project
			WHERE projectKey = $1
		)
	`
	if _, err := db.Exec(query, key); err != nil {
		return err
	}

	query = `DELETE FROM environment_group
		  USING environment, project, "group"
		  WHERE environment.id = environment_group.environment_id AND environment.project_id = project.id AND "group".id = environment_group.group_id
		  AND environment.name = $1 AND  project.projectKey = $2 AND "group".name = $3`
	_, err := db.Exec(query, envName, key, groupName)
	return err
}

func deleteGroupEnvironmentByGroup(db gorp.SqlExecutor, group *sdk.Group) error {
	query := `DELETE FROM environment_group WHERE group_id=$1`
	_, err := db.Exec(query, group.ID)
	return err
}
