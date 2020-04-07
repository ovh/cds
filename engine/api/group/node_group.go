package group

import (
	"database/sql"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

// LoadGroupsByNode retrieves all groups related to a node
func LoadGroupsByNode(db gorp.SqlExecutor, nodeID int64) ([]sdk.GroupPermission, error) {
	query := `SELECT "group".id,"group".name,workflow_node_group.role
		FROM "group"
			JOIN project_group ON "group".id = project_group.group_id
			JOIN workflow_perm ON workflow_perm.project_group_id = project_group.id
	 		JOIN workflow_node_group ON workflow_node_group.workflow_group_id = workflow_perm.id
		WHERE workflow_node_group.workflow_node_id = $1
		ORDER BY "group".name ASC`

	rows, err := db.Query(query, nodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}
	defer rows.Close()

	var groups []sdk.GroupPermission
	for rows.Next() {
		var group sdk.Group
		var perm int
		if err := rows.Scan(&group.ID, &group.Name, &perm); err != nil {
			return groups, sdk.WithStack(err)
		}
		groups = append(groups, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return groups, nil
}

// InsertGroupsInNode Link the given groups and the given environment
func InsertGroupsInNode(db gorp.SqlExecutor, groupPermission []sdk.GroupPermission, nodeID int64) error {
	for _, g := range groupPermission {
		if err := insertGroupInNode(db, nodeID, g.Group.ID, g.Permission); err != nil {
			return sdk.WrapError(err, "unable to insert group %d %s on node %d ", g.Group.ID, g.Group.Name, nodeID)
		}
	}
	return nil
}

// insertGroupInNode add permissions on Node to Group
func insertGroupInNode(db gorp.SqlExecutor, nodeID, groupID int64, role int) error {
	// avoid insert default env
	if sdk.DefaultEnv.ID == nodeID {
		return nil
	}
	query := `INSERT INTO workflow_node_group (workflow_node_id, workflow_group_id, role)
		VALUES(
			$1,
			(SELECT workflow_perm.id
			FROM workflow_perm
				JOIN project_group ON project_group.id = workflow_perm.project_group_id
				JOIN w_node ON w_node.workflow_id = workflow_perm.workflow_id
			WHERE w_node.id = $1 AND project_group.group_id = $2),
			$3
		)`
	if _, err := db.Exec(query, nodeID, groupID, role); err != nil {
		if strings.Contains(err.Error(), `null value in column "workflow_group_id"`) {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "cannot add this group on workflow node because it isn't in the workflow groups : %v", err)
		}
		return sdk.WithStack(err)
	}

	return nil
}

// DeleteAllGroupFromNode remove all groups from the given node id
func DeleteAllGroupFromNode(db gorp.SqlExecutor, nodeID int64) error {
	//Delete association
	query := "DELETE FROM workflow_node_group WHERE workflow_node_id = $1"
	_, err := db.Exec(query, nodeID)
	return sdk.WithStack(err)
}
