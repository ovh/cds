package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadWorkflowByGroup loads all workflows where group has access
func LoadWorkflowByGroup(db gorp.SqlExecutor, groupID int64) ([]sdk.WorkflowGroup, error) {
	res := []sdk.WorkflowGroup{}
	query := `SELECT project.projectKey,
			 		 workflow.id,
	                 workflow.name,
	                 workflow_group.role
	          FROM workflow
	          JOIN workflow_group ON workflow_group.workflow_id = workflow.id
	 	  JOIN project ON workflow.project_id = project.id
	 	  WHERE workflow_group.group_id = $1
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

// AddGroup Add permission on the given workflow for the given group
func AddGroup(db gorp.SqlExecutor, w *sdk.Workflow, gp sdk.GroupPermission) error {
	query := `INSERT INTO workflow_group (group_id, workflow_id, role) VALUES ($1, $2, $3)`
	if _, err := db.Exec(query, gp.Group.ID, w.ID, gp.Permission); err != nil {
		return sdk.WrapError(err, "AddGroup")
	}
	w.Groups = append(w.Groups, gp)
	return nil
}

// UpdateGroup  update group permission for the given group on the current workflow
func UpdateGroup(db gorp.SqlExecutor, w *sdk.Workflow, gp sdk.GroupPermission) error {
	query := `UPDATE  workflow_group SET role = $1 WHERE workflow_id = $2 AND group_id = $3`
	if _, err := db.Exec(query, gp.Permission, w.ID, gp.Group.ID); err != nil {
		return sdk.WrapError(err, "UpdateGroup")
	}

	for i := range w.Groups {
		g := &w.Groups[i]
		if g.Group.Name == gp.Group.Name {
			g.Permission = gp.Permission
		}
	}

	ok, err := checkAtLeastOneGroupWithWriteRoleOnWorkflow(db, w.ID)
	if err != nil {
		return sdk.WrapError(err, "UpdateGroup")
	}
	if !ok {
		return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "UpdateGroup")
	}
	return nil
}

func upsertAllGroups(db gorp.SqlExecutor, w *sdk.Workflow, gps []sdk.GroupPermission) error {
	queryD := `delete from workflow_group where workflow_id = $1`
	if _, err := db.Exec(queryD, w.ID); err != nil {
		return sdk.WrapError(err, "upsertAllGroups")
	}

	for _, gp := range gps {
		query := `INSERT INTO workflow_group (group_id, workflow_id, role) VALUES ($1, $2, $3)`
		if _, err := db.Exec(query, gp.Group.ID, w.ID, gp.Permission); err != nil {
			return sdk.WrapError(err, "upsertAllGroups> unable to insert group_id=%d workflow_id=%d role=%d", gp.Group.ID, w.ID, gp.Permission)
		}
	}

	ok, err := checkAtLeastOneGroupWithWriteRoleOnWorkflow(db, w.ID)
	if err != nil {
		return sdk.WrapError(err, "upsertAllGroups")
	}
	if !ok {
		return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "upsertAllGroups")
	}
	return nil
}

// DeleteGroup remove group permission on the given workflow
func DeleteGroup(db gorp.SqlExecutor, w *sdk.Workflow, groupID int64, index int) error {
	query := `DELETE FROM  workflow_group WHERE workflow_id = $1 AND group_id = $2`
	if _, err := db.Exec(query, w.ID, groupID); err != nil {
		return sdk.WrapError(err, "DeleteGroup")
	}
	w.Groups = append(w.Groups[:index], w.Groups[index+1:]...)

	ok, err := checkAtLeastOneGroupWithWriteRoleOnWorkflow(db, w.ID)
	if err != nil {
		return sdk.WrapError(err, "DeleteGroup")
	}
	if !ok {
		return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "DeleteGroup")
	}
	return nil
}

func checkAtLeastOneGroupWithWriteRoleOnWorkflow(db gorp.SqlExecutor, wID int64) (bool, error) {
	query := `select count(group_id) from workflow_group where workflow_id = $1 and role = $2`
	nb, err := db.SelectInt(query, wID, 7)
	if err != nil {
		return false, sdk.WrapError(err, "CheckAtLeastOneGroupWithWriteRoleOnWorkflow")
	}
	return nb > 0, err
}

func loadWorkflowGroups(db gorp.SqlExecutor, w sdk.Workflow) ([]sdk.GroupPermission, error) {
	wgs := []sdk.GroupPermission{}

	query := `SELECT "group".id, "group".name, workflow_group.role FROM "group"
	 		  JOIN workflow_group ON workflow_group.group_id = "group".id
	 		  WHERE workflow_group.workflow_id = $1 ORDER BY "group".name ASC`
	rows, errq := db.Query(query, w.ID)
	if errq != nil {
		if errq == sql.ErrNoRows {
			return wgs, nil
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
		wgs = append(wgs, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return wgs, nil
}

// ByGroupID List workflow that use the given group
func ByGroupID(db gorp.SqlExecutor, key string, groupID int64) ([]string, error) {
	query := `
		SELECT workflow.name  FROM workflow_group
		JOIN workflow ON workflow.id = workflow_group.workflow_id
		JOIN project ON project.id = workflow.project_id
		WHERE project.projectkey = $1 AND workflow_group.group_id = $2
	`
	wsName := make([]string, 0)
	rows, err := db.Query(query, key, groupID)
	if err != nil {
		return nil, sdk.WrapError(err, "group.WorkflowByGroupID> Unable to list environment")
	}
	defer rows.Close()
	for rows.Next() {
		var env string
		if err := rows.Scan(&env); err != nil {
			return nil, sdk.WrapError(err, "group.WorkflowByGroupID> Unable to scan")
		}
		wsName = append(wsName, env)
	}
	return wsName, nil
}
