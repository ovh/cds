package group

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadAllPipelineGroupByRole load all group for the given pipeline and role
func LoadAllPipelineGroupByRole(db gorp.SqlExecutor, pipelineID int64, role int) ([]sdk.GroupPermission, error) {
	groupsPermission := []sdk.GroupPermission{}
	query := `
		SELECT pipeline_group.group_id, pipeline_group.role
		FROM pipeline_group
		JOIN pipeline ON pipeline_group.pipeline_id = pipeline.id
		WHERE pipeline.id = $1 AND role = $2;
	`
	rows, err := db.Query(query, pipelineID, role)
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

// InsertGroupsInPipeline Link the given groups and the given pipeline
func InsertGroupsInPipeline(db gorp.SqlExecutor, groupPermission []sdk.GroupPermission, pipelineID int64) error {
	for _, g := range groupPermission {
		err := InsertGroupInPipeline(db, pipelineID, g.Group.ID, g.Permission)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkAtLeastOneGroupWithWriteRoleOnPipeline(db gorp.SqlExecutor, pipID int64) (bool, error) {
	query := `select count(group_id) from pipeline_group where pipeline_id = $1 and role = $2`
	nb, err := db.SelectInt(query, pipID, 7)
	if err != nil {
		return false, sdk.WrapError(err, "CheckAtLeastOneGroupWithWriteRoleOnPipeline")
	}
	return nb > 0, err
}

// InsertGroupInPipeline add permissions on Pipeline to Group
func InsertGroupInPipeline(db gorp.SqlExecutor, pipelineID, groupID int64, role int) error {
	query := `INSERT INTO pipeline_group (pipeline_id, group_id,role) VALUES($1,$2,$3)`
	_, err := db.Exec(query, pipelineID, groupID, role)
	return err
}

// UpdateGroupRoleInPipeline update permission on pipeline
func UpdateGroupRoleInPipeline(db gorp.SqlExecutor, pipelineID, groupID int64, role int) error {
	query := `UPDATE pipeline_group SET role=$1 WHERE pipeline_id=$2 AND group_id=$3`
	if _, err := db.Exec(query, role, pipelineID, groupID); err != nil {
		return sdk.WrapError(err, "UpdateGroupRoleInPipeline")
	}
	return nil
}

// DeleteGroupFromPipeline removes access to pipeline to group members
func DeleteGroupFromPipeline(db gorp.SqlExecutor, pipelineID, groupID int64) error {
	query := `DELETE FROM pipeline_group WHERE pipeline_id=$1 AND group_id=$2`
	if _, err := db.Exec(query, pipelineID, groupID); err != nil {
		return sdk.WrapError(err, "DeleteGroupFromPipeline")
	}

	ok, err := checkAtLeastOneGroupWithWriteRoleOnPipeline(db, pipelineID)
	if err != nil {
		return sdk.WrapError(err, "DeleteGroupFromPipeline")
	}
	if !ok {
		return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "DeleteGroupFromPipeline")
	}

	return err
}

// DeleteAllGroupFromPipeline Delete all groups from the given pipeline
func DeleteAllGroupFromPipeline(db gorp.SqlExecutor, pipelineID int64) error {
	query := `DELETE FROM pipeline_group WHERE pipeline_id=$1 `
	_, err := db.Exec(query, pipelineID)
	return err
}

// CheckGroupInPipeline checks if group has access to pipeline
func CheckGroupInPipeline(db gorp.SqlExecutor, pipelineID, groupID int64) (bool, error) {
	query := `SELECT COUNT(group_id) FROM pipeline_group WHERE pipeline_id = $1 AND group_id = $2`

	var nb int64
	err := db.QueryRow(query, pipelineID, groupID).Scan(&nb)
	if err != nil {
		return false, err
	}
	if nb != 0 {
		return true, nil
	}
	return false, nil
}

// LoadRoleGroupInPipeline load role from group linked to the pipeline
func LoadRoleGroupInPipeline(db gorp.SqlExecutor, pipelineID, groupID int64) (int64, error) {
	query := `SELECT role FROM pipeline_group WHERE pipeline_id = $1 AND group_id = $2`

	var nb int64
	role, err := db.SelectInt(query, pipelineID, groupID)
	if err != nil {
		return role, err
	}
	if nb != 0 {
		return role, nil
	}
	return role, nil
}

func deleteGroupPipelineByGroup(db gorp.SqlExecutor, group *sdk.Group) error {
	pipelineIDs := []int64{}
	if _, err := db.Select(&pipelineIDs, "SELECT pipeline_id from pipeline_group where group_id = $1", group.ID); err != nil && err != sql.ErrNoRows {
		return sdk.WrapError(err, "deleteGroupPipelineByGroup")
	}

	query := `DELETE FROM pipeline_group WHERE group_id=$1`
	if _, err := db.Exec(query, group.ID); err != nil {
		return sdk.WrapError(err, "deleteGroupPipelineByGroup")
	}

	for _, id := range pipelineIDs {
		ok, err := checkAtLeastOneGroupWithWriteRoleOnPipeline(db, id)
		if err != nil {
			return sdk.WrapError(err, "deleteGroupPipelineByGroup")
		}
		if !ok {
			return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "deleteGroupPipelineByGroup")
		}
	}

	return nil
}
