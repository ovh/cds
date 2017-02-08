package group

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadAllProjectGroupByRole load all group for the given project and role
func LoadAllProjectGroupByRole(db gorp.SqlExecutor, projectID int64, role int) ([]sdk.GroupPermission, error) {
	groupsPermission := []sdk.GroupPermission{}
	query := `
		SELECT project_group.group_id, project_group.role
		FROM project_group
		JOIN project ON project_group.project_id = project.id
		WHERE project.id = $1 AND role = $2;
	`
	rows, err := db.Query(query, projectID, role)
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

// DeleteGroupFromProject  Delete the group from the given project
func DeleteGroupFromProject(db gorp.SqlExecutor, projectID, groupID int64) error {
	query := `DELETE FROM project_group WHERE project_id=$1 AND group_id=$2`
	_, err := db.Exec(query, projectID, groupID)
	return err
}

// UpdateGroupRoleInProject Update group role for the given project
func UpdateGroupRoleInProject(db gorp.SqlExecutor, projectID, groupID int64, role int) error {
	query := `UPDATE project_group SET role=$1 WHERE project_id=$2 AND group_id=$3`
	_, err := db.Exec(query, role, projectID, groupID)
	return err
}

// InsertGroupInProject Attach a group to a project
func InsertGroupInProject(db gorp.SqlExecutor, projectID, groupID int64, role int) error {
	query := `INSERT INTO project_group (project_id, group_id,role) VALUES($1,$2,$3)`
	_, err := db.Exec(query, projectID, groupID, role)
	return err
}

// DeleteGroupProjectByProject removes group associated with project
func DeleteGroupProjectByProject(db gorp.SqlExecutor, projectID int64) error {
	query := `DELETE FROM project_group WHERE project_id=$1`
	_, err := db.Exec(query, projectID)
	if err != nil {
		return err
	}
	// Update project
	query = `
		UPDATE project 
		SET last_modified = current_timestamp
		WHERE id=$1
	`
	_, err = db.Exec(query, projectID)
	return err
}

func deleteGroupProjectByGroup(db gorp.SqlExecutor, group *sdk.Group) error {
	query := `DELETE FROM project_group WHERE group_id=$1`
	_, err := db.Exec(query, group.ID)
	if err != nil {
		return err
	}
	// Update project
	query = `
		UPDATE project 
		SET last_modified = current_timestamp
		WHERE id in (
			select project_id from project_group where group_id=$1
		)
	`
	_, err = db.Exec(query, group.ID)
	return err
}

// CheckGroupInProject  Check if the group is already attached to the project
func CheckGroupInProject(db gorp.SqlExecutor, projectID, groupID int64) (bool, error) {
	query := `SELECT COUNT(group_id) FROM project_group WHERE project_id = $1 AND group_id = $2`

	var nb int64
	err := db.QueryRow(query, projectID, groupID).Scan(&nb)
	if err != nil {
		return false, err
	}
	if nb != 0 {
		return true, nil
	}
	return false, nil
}
