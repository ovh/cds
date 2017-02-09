package group

import (
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

// InsertGroupInPipeline add permissions on Pipeline to Group
func InsertGroupInPipeline(db gorp.SqlExecutor, pipelineID, groupID int64, role int) error {
	query := `INSERT INTO pipeline_group (pipeline_id, group_id,role) VALUES($1,$2,$3)`
	_, err := db.Exec(query, pipelineID, groupID, role)
	return err
}

// UpdateGroupRoleInPipeline update permission on pipeline
func UpdateGroupRoleInPipeline(db gorp.SqlExecutor, pipelineID, groupID int64, role int) error {
	query := `UPDATE pipeline_group SET role=$1 WHERE pipeline_id=$2 AND group_id=$3`
	_, err := db.Exec(query, role, pipelineID, groupID)
	return err
}

// DeleteGroupFromPipeline removes access to pipeline to group members
func DeleteGroupFromPipeline(db gorp.SqlExecutor, pipelineID, groupID int64) error {
	query := `DELETE FROM pipeline_group WHERE pipeline_id=$1 AND group_id=$2`
	_, err := db.Exec(query, pipelineID, groupID)
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

func deleteGroupPipelineByGroup(db gorp.SqlExecutor, group *sdk.Group) error {
	query := `DELETE FROM pipeline_group WHERE group_id=$1`
	_, err := db.Exec(query, group.ID)
	return err
}
