package group

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func checkAtLeastOneGroupWithWriteRoleOnProject(db gorp.SqlExecutor, projectID int64) (bool, error) {
	query := `select count(group_id) from project_group where project_id = $1 and role = $2`
	nb, err := db.SelectInt(query, projectID, 7)
	if err != nil {
		return false, sdk.WrapError(err, "CheckAtLeastOneGroupWithWriteRoleOnProject")
	}
	return nb > 0, err
}

// DeleteGroupFromProject  Delete the group from the given project
func DeleteGroupFromProject(db gorp.SqlExecutor, projectID, groupID int64) error {
	query := `DELETE FROM project_group WHERE project_id=$1 AND group_id=$2`
	if _, err := db.Exec(query, projectID, groupID); err != nil {
		return sdk.WrapError(err, "DeleteGroupFromProject")
	}

	ok, err := checkAtLeastOneGroupWithWriteRoleOnProject(db, projectID)
	if err != nil {
		return sdk.WrapError(err, "DeleteGroupFromProject")
	}
	if !ok {
		return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "DeleteGroupFromProject")
	}

	return nil
}

// DeleteAllGroupFromProject Delete all groups from the given project ID
func DeleteAllGroupFromProject(db gorp.SqlExecutor, projectID int64) error {
	query := `DELETE FROM project_group WHERE project_id=$1 `
	_, err := db.Exec(query, projectID)
	return err
}

// UpdateGroupRoleInProject Update group role for the given project
func UpdateGroupRoleInProject(db gorp.SqlExecutor, projectID, groupID int64, role int) error {
	query := `UPDATE project_group SET role=$1 WHERE project_id=$2 AND group_id=$3`
	if _, err := db.Exec(query, role, projectID, groupID); err != nil {
		return sdk.WrapError(err, "UpdateGroupRoleInProject")
	}

	ok, err := checkAtLeastOneGroupWithWriteRoleOnProject(db, projectID)
	if err != nil {
		return sdk.WrapError(err, "UpdateGroupRoleInProject (checkAtLeastOneGroupWithWriteRoleOnProject)")
	}
	if !ok {
		return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "UpdateGroupRoleInProject")
	}

	return nil
}

// InsertGroupInProject Attach a group to a project
func InsertGroupInProject(db gorp.SqlExecutor, projectID, groupID int64, role int) error {
	query := `INSERT INTO project_group (project_id, group_id, role) VALUES($1,$2,$3)`
	_, err := db.Exec(query, projectID, groupID, role)
	return err
}

// DeleteGroupProjectByProject removes group associated with project
// Only use by delete project
func DeleteGroupProjectByProject(db gorp.SqlExecutor, projectID int64) error {
	query := `DELETE FROM project_group WHERE project_id=$1`
	_, err := db.Exec(query, projectID)
	return err
}

func deleteGroupProjectByGroup(db gorp.SqlExecutor, group *sdk.Group) error {
	projectIDs := []int64{}
	if _, err := db.Select(&projectIDs, "SELECT project_id from project_group where group_id = $1", group.ID); err != nil && err != sql.ErrNoRows {
		return sdk.WrapError(err, "deleteGroupProjectByGroup")
	}

	query := `DELETE FROM project_group WHERE group_id=$1`
	if _, err := db.Exec(query, group.ID); err != nil {
		return sdk.WrapError(err, "deleteGroupProjectByGroup")
	}

	for _, id := range projectIDs {
		ok, err := checkAtLeastOneGroupWithWriteRoleOnProject(db, id)
		if err != nil {
			return sdk.WrapError(err, "deleteGroupProjectByGroup")
		}
		if !ok {
			return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "deleteGroupProjectByGroup")
		}
	}

	return nil
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
