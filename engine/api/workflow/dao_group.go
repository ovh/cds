package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// ByGroupID List workflow that use the given group
func ByGroupID(db gorp.SqlExecutor, key string, groupID int64) ([]string, error) {
	query := `
		SELECT workflow.name FROM workflow_perm
			JOIN project_group ON project_group.id = workflow_perm.project_group_id
			JOIN workflow ON workflow.id = workflow_perm.workflow_id
			JOIN project ON project.id = workflow.project_id
		WHERE project.projectkey = $1 AND project_group.group_id = $2
	`
	wsName := make([]string, 0)
	rows, err := db.Query(query, key, groupID)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to list environment")
	}
	defer rows.Close()
	for rows.Next() {
		var env string
		if err := rows.Scan(&env); err != nil {
			return nil, sdk.WrapError(err, "Unable to scan")
		}
		wsName = append(wsName, env)
	}
	return wsName, nil
}
