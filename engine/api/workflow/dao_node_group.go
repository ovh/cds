package workflow

import (
	"database/sql"

	"github.com/ovh/cds/sdk"

	"github.com/go-gorp/gorp"
)

func loadWorkflowNodeGroups(db gorp.SqlExecutor, workflowNodeID int64) ([]sdk.GroupPermission, error) {
	var wNodegrs []sdk.GroupPermission

	query := `SELECT "group".id, "group".name, workflow_node_group.role
		FROM "group"
	 		JOIN workflow_node_group ON workflow_node_group.group_id = "group".id
		 WHERE workflow_node_group.workflow_node_id = $1
		 ORDER BY "group".name ASC`
	rows, errq := db.Query(query, workflowNodeID)
	if errq != nil {
		if errq == sql.ErrNoRows {
			return wNodegrs, nil
		}
		return nil, errq
	}
	defer rows.Close()

	for rows.Next() {
		var group sdk.Group
		var perm int
		if err := rows.Scan(&group.ID, &group.Name, &perm); err != nil {
			return nil, err
		}
		wNodegrs = append(wNodegrs, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return wNodegrs, nil
}
