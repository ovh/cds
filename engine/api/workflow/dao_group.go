package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadWorkflowByGroup loads all workflows where group has access
func LoadWorkflowByGroup(db gorp.SqlExecutor, groupID int64) ([]sdk.WorkflowGroup, error) {
	res := []sdk.WorkflowGroup{}
	query := `SELECT project.projectKey,
			 	workflow.id,
             	workflow.name,
             	workflow_perm.role
	        FROM workflow
	          	JOIN workflow_perm ON workflow_perm.workflow_id = workflow.id
	          	JOIN project_group ON project_group.id = workflow_perm.project_group_id
	 	  		JOIN project ON workflow.project_id = project.id
	 	 	WHERE project_group.group_id = $1
	 	  	ORDER BY workflow.name ASC`
	rows, err := db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var w sdk.Workflow
		var perm int
		if err := rows.Scan(&w.ProjectKey, &w.ID, &w.Name, &perm); err != nil {
			return nil, err
		}
		res = append(res, sdk.WorkflowGroup{
			Workflow:   w,
			Permission: perm,
		})
	}
	return res, nil
}

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
