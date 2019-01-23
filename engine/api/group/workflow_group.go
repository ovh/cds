package group

import (
	"database/sql"

	"github.com/go-gorp/gorp"
)

// LoadRoleGroupInWorkflow load role from group linked to the workflow
func LoadRoleGroupInWorkflow(db gorp.SqlExecutor, workflowID, groupID int64) (int, error) {
	query := `SELECT workflow_perm.role
	FROM workflow_perm
		JOIN project_group ON workflow_perm.project_group_id = project_group.id
	WHERE workflow_perm.workflow_id = $1 AND project_group.group_id = $2`

	role, err := db.SelectInt(query, workflowID, groupID)
	if err != nil {
		return int(role), err
	}
	return int(role), nil
}

// LoadRoleGroupInWorkflowNode load role from group linked to the workflow node
func LoadRoleGroupInWorkflowNode(db gorp.SqlExecutor, nodeID, groupID int64) (int, error) {
	queryNode := `SELECT workflow_node_group.role
	FROM workflow_node_group
		JOIN workflow_perm ON workflow_perm.id = workflow_node_group.workflow_group_id
		JOIN project_group ON workflow_perm.project_group_id = project_group.id
	WHERE workflow_node_group.workflow_node_id = $1 AND project_group.group_id = $2`

	role, err := db.SelectInt(queryNode, nodeID, groupID)
	if err != nil && err != sql.ErrNoRows {
		return int(role), err
	}

	query := `SELECT workflow_perm.role
	FROM workflow_perm
		JOIN project_group ON workflow_perm.project_group_id = project_group.id
	WHERE workflow_perm.workflow_id = $1 AND project_group.group_id = $2`

	role, err = db.SelectInt(query, nodeID, groupID)
	if err != nil {
		return int(role), err
	}

	return int(role), nil
}
