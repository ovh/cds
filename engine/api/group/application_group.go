package group

import (
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

// CheckGroupInApplication  Check if the group is already attached to the application
func CheckGroupInApplication(db gorp.SqlExecutor, applicationID, groupID int64) (bool, error) {
	query := `SELECT COUNT(group_id) FROM application_group WHERE application_id = $1 AND group_id = $2`

	var nb int64
	err := db.QueryRow(query, applicationID, groupID).Scan(&nb)
	if err != nil {
		return false, err
	}
	return (nb != 0), nil
}

// InsertGroupInApplication add permissions on Application to Group
func InsertGroupInApplication(db gorp.SqlExecutor, applicationID, groupID int64, role int) error {
	query := `INSERT INTO application_group (application_id, group_id,role) VALUES($1,$2,$3)`
	_, err := db.Exec(query, applicationID, groupID, role)
	return err
}

// UpdateGroupRoleInApplication update permission on application
func UpdateGroupRoleInApplication(db gorp.SqlExecutor, key, appName, groupName string, role int) error {
	query := `UPDATE application_group
	          SET role=$1
	          FROM application, project, "group"
	          WHERE application.id = application_id AND application.project_id = project.id AND "group".id = group_id
	          AND application.name = $2 AND  project.projectKey = $3 AND "group".name = $4 `
	_, err := db.Exec(query, role, appName, key, groupName)
	return err
}

// DeleteAllGroupFromApplication remove all group from the given application
func DeleteAllGroupFromApplication(db gorp.SqlExecutor, applicationID int64) error {
	query := `DELETE FROM application_group
		  WHERE application_id=$1`
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
	query := `DELETE FROM application_group WHERE group_id=$1`
	_, err := db.Exec(query, group.ID)
	return err
}
