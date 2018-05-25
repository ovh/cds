package group

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadGroupsByEnvironment retrieves all groups related to an env
func LoadGroupsByEnvironment(db gorp.SqlExecutor, envID int64) ([]sdk.GroupPermission, error) {
	query := `SELECT "group".id,"group".name,environment_group.role FROM "group"
	 		  JOIN environment_group ON environment_group.group_id = "group".id
			  WHERE environment_group.environment_id = $1
	 		  ORDER BY "group".name ASC`

	rows, err := db.Query(query, envID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	groups := []sdk.GroupPermission{}
	for rows.Next() {
		var group sdk.Group
		var perm int
		if err := rows.Scan(&group.ID, &group.Name, &perm); err != nil {
			return groups, err
		}
		groups = append(groups, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return groups, nil
}

// EnvironmentsByGroupID List environment that use the given group
func EnvironmentsByGroupID(db gorp.SqlExecutor, key string, groupID int64) ([]string, error) {
	query := `
		SELECT environment.name  FROM environment_group
		JOIN environment ON environment.id = environment_group.environment_id
		JOIN project ON project.id = environment.project_id
		WHERE project.projectkey = $1 AND environment_group.group_id = $2
	`
	envsName := make([]string, 0)
	rows, err := db.Query(query, key, groupID)
	if err != nil {
		return nil, sdk.WrapError(err, "group.EnvironmentsByGroupID> Unable to list environment")
	}
	defer rows.Close()
	for rows.Next() {
		var env string
		if err := rows.Scan(&env); err != nil {
			return nil, sdk.WrapError(err, "group.EnvironmentsByGroupID> Unable to scan")
		}
		envsName = append(envsName, env)
	}
	return envsName, nil
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

// checkAtLeastOneGroupWithWriteRoleOnEnvironment is clear enough i think
func checkAtLeastOneGroupWithWriteRoleOnEnvironment(db gorp.SqlExecutor, envID int64) (bool, error) {
	// no check for default env
	if sdk.DefaultEnv.ID == envID {
		return true, nil
	}

	query := `select count(group_id) from environment_group where environment_id = $1 and role = $2`
	nb, err := db.SelectInt(query, envID, 7)
	if err != nil {
		return false, sdk.WrapError(err, "CheckAtLeastOneGroupWithWriteRoleOnEnvironment")
	}
	return nb > 0, err
}

// InsertGroupInEnvironment add permissions on Environment to Group
func InsertGroupInEnvironment(db gorp.SqlExecutor, environmentID, groupID int64, role int) error {
	// avoid insert default env
	if sdk.DefaultEnv.ID == environmentID {
		return nil
	}
	query := `INSERT INTO environment_group (environment_id, group_id,role) VALUES($1,$2,$3)`
	if _, err := db.Exec(query, environmentID, groupID, role); err != nil {
		return sdk.WrapError(err, "InsertGroupInEnvironment")
	}

	return nil
}

// InsertGroupsInEnvironment Link the given groups and the given environment
func InsertGroupsInEnvironment(db gorp.SqlExecutor, groupPermission []sdk.GroupPermission, envID int64) error {
	for _, g := range groupPermission {
		if err := InsertGroupInEnvironment(db, envID, g.Group.ID, g.Permission); err != nil {
			return sdk.WrapError(err, "InsertGroupsInEnvironment> unable to insert group %d %s on env %d ", g.Group.ID, g.Group.Name, envID)
		}
	}
	return nil
}

// UpdateGroupRoleInEnvironment update permission on environment
func UpdateGroupRoleInEnvironment(db gorp.SqlExecutor, environmentID, groupID int64, role int) error {
	query := `UPDATE environment_group
	          SET role=$1
	          WHERE environment_id = $2 and group_id = $3`
	if _, err := db.Exec(query, role, environmentID, groupID); err != nil {
		return err
	}

	ok, err := checkAtLeastOneGroupWithWriteRoleOnEnvironment(db, environmentID)
	if err != nil {
		return sdk.WrapError(err, "UpdateGroupRoleInEnvironment")
	}
	if !ok {
		return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "UpdateGroupRoleInEnvironment")
	}

	return nil
}

// DeleteAllGroupFromEnvironment remove all group from the given environment
// Deprecated
func DeleteAllGroupFromEnvironment(db gorp.SqlExecutor, environmentID int64) error {
	//Delete association
	query := `DELETE FROM environment_group
		  WHERE environment_id=$1`
	_, err := db.Exec(query, environmentID)
	return err
}

// DeleteGroupFromEnvironment removes access to environment to group members
func DeleteGroupFromEnvironment(db gorp.SqlExecutor, envID, groupID int64) error {
	query := `DELETE FROM environment_group
		  WHERE environment_id = $1 
		  AND group_id = $2`
	if _, err := db.Exec(query, envID, groupID); err != nil {
		return sdk.WrapError(err, "DeleteGroupFromEnvironment")
	}
	ok, err := checkAtLeastOneGroupWithWriteRoleOnEnvironment(db, envID)
	if err != nil {
		return sdk.WrapError(err, "deleteGroupEnvironmentByGroup")
	}
	if !ok {
		return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "deleteGroupEnvironmentByGroup")
	}
	return nil
}

func deleteGroupEnvironmentByGroup(db gorp.SqlExecutor, group *sdk.Group) error {
	envIDs := []int64{}
	if _, err := db.Select(&envIDs, "SELECT environment_id from environment_group where group_id = $1", group.ID); err != nil && err != sql.ErrNoRows {
		return sdk.WrapError(err, "deleteGroupEnvironmentByGroup")
	}

	query := `DELETE FROM environment_group WHERE group_id=$1`
	if _, err := db.Exec(query, group.ID); err != nil {
		return sdk.WrapError(err, "deleteGroupEnvironmentByGroup")
	}

	for _, id := range envIDs {
		ok, err := checkAtLeastOneGroupWithWriteRoleOnEnvironment(db, id)
		if err != nil {
			return sdk.WrapError(err, "deleteGroupEnvironmentByGroup")
		}
		if !ok {
			return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "deleteGroupEnvironmentByGroup")
		}
	}

	return nil
}
