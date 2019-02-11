package group

import (
	"database/sql"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
)

// LoadRoleGroupInWorkflow load role from group linked to the workflow
func LoadRoleGroupInWorkflow(db gorp.SqlExecutor, workflowID, groupID int64) (int, error) {
	query := `SELECT workflow_perm.role
	FROM workflow_perm
		JOIN project_group ON workflow_perm.project_group_id = project_group.id
	WHERE workflow_perm.workflow_id = $1 AND project_group.group_id = $2`

	role, err := db.SelectInt(query, workflowID, groupID)
	if err != nil {
		return int(role), sdk.WithStack(err)
	}
	return int(role), nil
}

// ExistGroupInWorkflow return boolean to indicate if a group exist in this workflow
func ExistGroupInWorkflow(db gorp.SqlExecutor, workflowID, groupID int64) (bool, error) {
	query := `SELECT COUNT(workflow_perm.id)
	FROM workflow_perm
		JOIN project_group ON workflow_perm.project_group_id = project_group.id
	WHERE workflow_perm.workflow_id = $1 AND project_group.group_id = $2`

	count, err := db.SelectInt(query, workflowID, groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, sdk.WithStack(err)
	}
	return count > 0, nil
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
		return int(role), sdk.WithStack(err)
	}

	query := `SELECT workflow_perm.role
	FROM workflow_perm
		JOIN project_group ON workflow_perm.project_group_id = project_group.id
	WHERE workflow_perm.workflow_id = $1 AND project_group.group_id = $2`

	role, err = db.SelectInt(query, nodeID, groupID)
	if err != nil {
		return int(role), sdk.WithStack(err)
	}

	return int(role), nil
}

// AddWorkflowGroup Add permission on the given workflow for the given group
func AddWorkflowGroup(db gorp.SqlExecutor, w *sdk.Workflow, gp sdk.GroupPermission) error {
	projectGroupID, projectRole, err := LoadRoleGroupInProject(db, w.ProjectID, gp.Group.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return sdk.WrapError(sdk.ErrGroupNotFoundInProject, "cannot add this group on workflow because there isn't in the project groups : %v", err)
		}
		return sdk.WrapError(err, "Cannot load role for group %d in project %d", gp.Group.ID, w.ProjectID)
	}
	if projectRole == permission.PermissionReadWriteExecute && gp.Permission < projectRole {
		return sdk.ErrWorkflowPermInsufficient
	}

	query := `INSERT INTO workflow_perm (project_group_id, workflow_id, role)
	VALUES (
		$1,
		$2,
		$3
	)`
	if _, err := db.Exec(query, projectGroupID, w.ID, gp.Permission); err != nil {
		return err
	}
	w.Groups = append(w.Groups, gp)
	return nil
}

// UpdateWorkflowGroup  update group permission for the given group on the current workflow
func UpdateWorkflowGroup(db gorp.SqlExecutor, w *sdk.Workflow, gp sdk.GroupPermission) error {
	_, projectRole, err := LoadRoleGroupInProject(db, w.ProjectID, gp.Group.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return sdk.WrapError(sdk.ErrGroupNotFoundInProject, "cannot update this group on workflow because there isn't in the project groups : %v", err)
		}
		return sdk.WrapError(err, "Cannot load role for group %d in project %d", gp.Group.ID, w.ProjectID)
	}
	if projectRole == permission.PermissionReadWriteExecute && gp.Permission < projectRole {
		return sdk.ErrWorkflowPermInsufficient
	}

	query := `UPDATE workflow_perm
	SET role = $1
	FROM project_group
	WHERE project_group.id = workflow_perm.project_group_id AND workflow_perm.workflow_id = $2 AND project_group.group_id = $3`
	if _, err := db.Exec(query, gp.Permission, w.ID, gp.Group.ID); err != nil {
		return sdk.WithStack(err)
	}

	for i := range w.Groups {
		g := &w.Groups[i]
		if g.Group.Name == gp.Group.Name {
			g.Permission = gp.Permission
		}
	}

	ok, err := checkAtLeastOneGroupWithWriteRoleOnWorkflow(db, w.ID)
	if err != nil {
		return err
	}
	if !ok {
		return sdk.ErrLastGroupWithWriteRole
	}
	return nil
}

// UpsertAllWorkflowGroups upsert all groups in a workflow
func UpsertAllWorkflowGroups(db gorp.SqlExecutor, w *sdk.Workflow, gps []sdk.GroupPermission) error {
	for _, gp := range gps {
		if err := UpsertWorkflowGroup(db, w.ProjectID, w.ID, gp); err != nil {
			return err
		}
	}

	ok, err := checkAtLeastOneGroupWithWriteRoleOnWorkflow(db, w.ID)
	if err != nil {
		return sdk.WrapError(err, "U")
	}
	if !ok {
		return sdk.WrapError(sdk.ErrLastGroupWithWriteRole, "U")
	}
	return nil
}

// UpsertWorkflowGroup upsert a workflow group
func UpsertWorkflowGroup(db gorp.SqlExecutor, projectID, workflowID int64, gp sdk.GroupPermission) error {
	query := `INSERT INTO workflow_perm (project_group_id, workflow_id, role) 
			VALUES (
				(SELECT id FROM project_group WHERE project_group.project_id = $1 AND project_group.group_id = $2),
				$3,
				$4
			) ON CONFLICT DO NOTHING`
	if _, err := db.Exec(query, projectID, gp.Group.ID, workflowID, gp.Permission); err != nil {
		if strings.Contains(err.Error(), `null value in column "project_group_id"`) {
			return sdk.WrapError(sdk.ErrGroupNotFoundInProject, "cannot add this group on workflow because there isn't in the project groups : %v", err)
		}
		return sdk.WrapError(err, "unable to insert group_id=%d workflow_id=%d role=%d", gp.Group.ID, workflowID, gp.Permission)
	}

	return nil
}

// DeleteWorkflowGroup remove group permission on the given workflow
func DeleteWorkflowGroup(db gorp.SqlExecutor, w *sdk.Workflow, groupID int64, index int) error {
	query := `DELETE FROM workflow_perm
		USING project_group
	WHERE workflow_perm.project_group_id = project_group.id AND workflow_perm.workflow_id = $1 AND project_group.group_id = $2`
	if _, err := db.Exec(query, w.ID, groupID); err != nil {
		return sdk.WithStack(err)
	}

	ok, err := checkAtLeastOneGroupWithWriteRoleOnWorkflow(db, w.ID)
	if err != nil {
		return err
	}
	if !ok {
		return sdk.ErrLastGroupWithWriteRole
	}
	w.Groups = append(w.Groups[:index], w.Groups[index+1:]...)
	return nil
}

func checkAtLeastOneGroupWithWriteRoleOnWorkflow(db gorp.SqlExecutor, wID int64) (bool, error) {
	query := `select count(project_group_id) from workflow_perm where workflow_id = $1 and role = $2`
	nb, err := db.SelectInt(query, wID, 7)
	if err != nil {
		return false, sdk.WithStack(err)
	}
	return nb > 0, err
}

// LoadWorkflowGroups load groups for a workflow
func LoadWorkflowGroups(db gorp.SqlExecutor, workflowID int64) ([]sdk.GroupPermission, error) {
	wgs := []sdk.GroupPermission{}

	query := `SELECT "group".id, "group".name, workflow_perm.role
		FROM "group"
			JOIN project_group ON project_group.group_id = "group".id
			JOIN workflow_perm ON workflow_perm.project_group_id = project_group.id
		WHERE workflow_perm.workflow_id = $1
		ORDER BY "group".name ASC`
	rows, errq := db.Query(query, workflowID)
	if errq != nil {
		if errq == sql.ErrNoRows {
			return wgs, nil
		}
		return nil, sdk.WithStack(errq)
	}
	defer rows.Close()

	for rows.Next() {
		var group sdk.Group
		var perm int
		if err := rows.Scan(&group.ID, &group.Name, &perm); err != nil {
			return nil, sdk.WithStack(err)
		}
		wgs = append(wgs, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return wgs, nil
}
