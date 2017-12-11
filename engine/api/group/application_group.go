package group

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadAllApplicationGroupByRole load all group for the given application and role
func LoadAllApplicationGroupByRole(db gorp.SqlExecutor, applicationID int64, role int) ([]sdk.GroupPermission, error) {
	groupsPermission := []sdk.GroupPermission{}
	query := `
		SELECT application_group.group_id, application_group.role
		FROM application_group
		JOIN application ON application_group.application_id = application.id
		WHERE application.id = $1 AND role = $2;
	`
	rows, err := db.Query(query, applicationID, role)
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

// LoadGroupsByApplication retrieves all groups related to project
func LoadGroupsByApplication(db gorp.SqlExecutor, appID int64) ([]sdk.GroupPermission, error) {
	query := `SELECT "group".id,"group".name,application_group.role FROM "group"
	 		  JOIN application_group ON application_group.group_id = "group".id
				JOIN application ON application.id = application_group.application_id
	 		  WHERE application.id = $1  ORDER BY "group".name ASC`

	rows, err := db.Query(query, appID)
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

// CheckGroupInApplication  Check if the group is already attached to the application
func CheckGroupInApplication(db gorp.SqlExecutor, applicationID, groupID int64) (bool, error) {
	query := `SELECT COUNT(group_id) FROM application_group WHERE application_id = $1 AND group_id = $2`

	var nb int64
	if err := db.QueryRow(query, applicationID, groupID).Scan(&nb); err != nil {
		return false, err
	}
	return (nb != 0), nil
}

func checkAtLeastOneGroupWithWriteRoleOnApplication(db gorp.SqlExecutor, appID int64) (bool, error) {
	query := `select count(group_id) from application_group where application_id = $1 and role = $2`
	nb, err := db.SelectInt(query, appID, 7)
	if err != nil {
		return false, sdk.WrapError(err, "checkAtLeastOneGroupWithWriteRoleOnApplication")
	}
	return nb > 0, err
}

// InsertGroupInApplication add permissions on Application to Group
func InsertGroupInApplication(db gorp.SqlExecutor, applicationID, groupID int64, role int) error {
	query := `INSERT INTO application_group (application_id, group_id,role) VALUES($1,$2,$3)`
	_, err := db.Exec(query, applicationID, groupID, role)
	return err
}

// UpdateGroupRoleInApplication update permission on application
func UpdateGroupRoleInApplication(db gorp.SqlExecutor, appID, groupID int64, role int) error {
	query := `UPDATE application_group
	          SET role=$1
			  WHERE application_id = $2
			  AND group_id = $3`
	_, err := db.Exec(query, role, appID, groupID)

	ok, err := checkAtLeastOneGroupWithWriteRoleOnApplication(db, appID)
	if err != nil {
		return sdk.WrapError(err, "UpdateGroupRoleInApplication")
	}
	if !ok {
		return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "UpdateGroupRoleInApplication")
	}

	return err
}

// DeleteAllGroupFromApplication remove all group from the given application
func DeleteAllGroupFromApplication(db gorp.SqlExecutor, applicationID int64) error {
	query := `DELETE FROM application_group WHERE application_id=$1`
	_, err := db.Exec(query, applicationID)
	return err
}

// DeleteGroupFromApplication removes access to application to group members
func DeleteGroupFromApplication(db gorp.SqlExecutor, key, appName, groupName string) error {
	query := `DELETE FROM application_group
		  USING application, project, "group"
		  WHERE application.id = application_group.application_id AND application.project_id = project.id AND "group".id = application_group.group_id
		  AND application.name = $1 AND  project.projectKey = $2 AND "group".name = $3`
	_, err := db.Exec(query, appName, key, groupName)
	return err
}

func deleteGroupApplicationByGroup(db gorp.SqlExecutor, group *sdk.Group) error {
	appIDs := []int64{}
	if _, err := db.Select(&appIDs, "SELECT application_id from application_group where group_id = $1", group.ID); err != nil && err != sql.ErrNoRows {
		return sdk.WrapError(err, "deleteGroupPipelineByGroup")
	}

	query := `DELETE FROM application_group WHERE group_id=$1`
	if _, err := db.Exec(query, group.ID); err != nil {
		return sdk.WrapError(err, "deleteGroupApplicationByGroup")
	}

	for _, id := range appIDs {
		ok, err := checkAtLeastOneGroupWithWriteRoleOnApplication(db, id)
		if err != nil {
			return sdk.WrapError(err, "deleteGroupApplicationByGroup")
		}
		if !ok {
			return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "deleteGroupApplicationByGroup")
		}
	}

	return nil
}
