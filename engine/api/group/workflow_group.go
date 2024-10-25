package group

import (
	"context"
	"database/sql"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/gorpmapper"
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

// AddWorkflowGroup Add permission on the given workflow for the given group
func AddWorkflowGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, w *sdk.Workflow, gp sdk.GroupPermission) error {
	link, err := LoadLinkGroupProjectForGroupIDAndProjectID(ctx, db, gp.Group.ID, w.ProjectID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return sdk.WithStack(sdk.ErrGroupNotFoundInProject)
		}
		return sdk.WrapError(err, "cannot load role for group %d in project %d", gp.Group.ID, w.ProjectID)
	}
	if link.Role == sdk.PermissionReadWriteExecute && gp.Permission < link.Role {
		return sdk.WithStack(sdk.ErrWorkflowPermInsufficient)
	}

	query := `INSERT INTO workflow_perm (project_group_id, workflow_id, role)	VALUES ($1,	$2,	$3)`
	if _, err := db.Exec(query, link.ID, w.ID, gp.Permission); err != nil {
		return sdk.WithStack(err)
	}
	w.Groups = append(w.Groups, gp)
	return nil
}

// UpdateWorkflowGroup  update group permission for the given group on the current workflow
func UpdateWorkflowGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, w *sdk.Workflow, gp sdk.GroupPermission) error {
	link, err := LoadLinkGroupProjectForGroupIDAndProjectID(ctx, db, gp.Group.ID, w.ProjectID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return sdk.WithStack(sdk.ErrGroupNotFoundInProject)
		}
		return sdk.WrapError(err, "cannot load role for group %d in project %d", gp.Group.ID, w.ProjectID)
	}
	if link.Role == sdk.PermissionReadWriteExecute && gp.Permission < link.Role {
		return sdk.WithStack(sdk.ErrWorkflowPermInsufficient)
	}

	query := "UPDATE workflow_perm SET role = $3 WHERE project_group_id = $1 AND workflow_id = $2"
	if _, err := db.Exec(query, link.ID, w.ID, gp.Permission); err != nil {
		return sdk.WithStack(err)
	}

	for i := range w.Groups {
		g := &w.Groups[i]
		if g.Group.Name == gp.Group.Name {
			g.Permission = gp.Permission
		}
	}

	if err := checkAtLeastOneRWXRoleOnWorkflow(db, w.ID); err != nil {
		return err
	}

	return nil
}

// UpsertAllWorkflowGroups upsert all groups in a workflow
func UpsertAllWorkflowGroups(ctx context.Context, db gorpmapper.SqlExecutorWithTx, w *sdk.Workflow, gps []sdk.GroupPermission) error {
	query := "DELETE FROM workflow_perm WHERE workflow_id = $1"
	if _, err := db.Exec(query, w.ID); err != nil {
		return sdk.WrapError(err, "unable to remove group permissions for workflow %d", w.ID)
	}

	for _, gp := range gps {
		link, err := LoadLinkGroupProjectForGroupIDAndProjectID(ctx, db, gp.Group.ID, w.ProjectID)
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				return sdk.WithStack(sdk.ErrGroupNotFoundInProject)
			}
			return sdk.WrapError(err, "cannot load role for group %d in project %d", gp.Group.ID, w.ProjectID)
		}
		if link.Role == sdk.PermissionReadWriteExecute && gp.Permission < link.Role {
			return sdk.WithStack(sdk.ErrWorkflowPermInsufficient)
		}

		if err := UpsertWorkflowGroup(db, w.ProjectID, w.ID, gp); err != nil {
			return err
		}
	}

	if err := checkAtLeastOneRWXRoleOnWorkflow(db, w.ID); err != nil {
		return err
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
			) ON CONFLICT (project_group_id, workflow_id) DO UPDATE SET role = $4`
	if _, err := db.Exec(query, projectID, gp.Group.ID, workflowID, gp.Permission); err != nil {
		if strings.Contains(err.Error(), `null value in column "project_group_id"`) {
			return sdk.WrapError(sdk.ErrNotFound, "cannot add this group on workflow because there isn't in the project groups : %v", err)
		}
		return sdk.WrapError(err, "unable to insert group_id=%d workflow_id=%d role=%d", gp.Group.ID, workflowID, gp.Permission)
	}
	return nil
}

// DeleteWorkflowGroup remove group permission on the given workflow
func DeleteWorkflowGroup(db gorp.SqlExecutor, w *sdk.Workflow, groupID int64, index int) error {
	query := `
    DELETE FROM workflow_perm
		USING project_group
    WHERE workflow_perm.project_group_id = project_group.id AND workflow_perm.workflow_id = $1 AND project_group.group_id = $2
  `
	if _, err := db.Exec(query, w.ID, groupID); err != nil {
		return sdk.WithStack(err)
	}

	if err := checkAtLeastOneRWXRoleOnWorkflow(db, w.ID); err != nil {
		return err
	}

	w.Groups = append(w.Groups[:index], w.Groups[index+1:]...)

	return nil
}

func checkAtLeastOneRWXRoleOnWorkflow(db gorp.SqlExecutor, wID int64) error {
	query := `select count(project_group_id) from workflow_perm where workflow_id = $1 and role = $2`
	nb, err := db.SelectInt(query, wID, sdk.PermissionReadWriteExecute)
	if err != nil {
		return sdk.WithStack(err)
	}
	if nb == 0 {
		return sdk.WithStack(sdk.ErrLastGroupWithWriteRole)
	}
	return nil
}

type LinkWorkflowGroupPermission struct {
	WorkflowID int64  `db:"workflow_id"`
	GroupID    int64  `db:"group_id"`
	GroupName  string `db:"group_name"`
	Role       int    `db:"role"`
}

// LoadWorkflowGroupsByWorkflowIDs returns a map with key: workflowID and value the slite of groups
func LoadWorkflowGroupsByWorkflowIDs(ctx context.Context, db gorp.SqlExecutor, workflowIDs []int64) (map[int64][]sdk.GroupPermission, error) {
	result := make(map[int64][]sdk.GroupPermission, len(workflowIDs))
	query := gorpmapping.NewQuery(`
    SELECT workflow_perm.workflow_id, "group".id as "group_id", "group".name as "group_name", workflow_perm.role
    FROM "group"
    JOIN project_group ON project_group.group_id = "group".id
    JOIN workflow_perm ON workflow_perm.project_group_id = project_group.id
    WHERE workflow_perm.workflow_id = ANY($1)
    ORDER BY workflow_perm.workflow_id, "group".name ASC
	`).Args(pq.Int64Array(workflowIDs))
	var dbResultSet = []LinkWorkflowGroupPermission{}
	if err := gorpmapping.GetAll(ctx, db, query, &dbResultSet); err != nil {
		return nil, err
	}

	var groupIDs []int64
	for i := range dbResultSet {
		groupIDs = append(groupIDs, dbResultSet[i].GroupID)
	}
	groupOrganizations, err := LoadGroupOrganizationsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return nil, err
	}
	mGrpOrgs := make(map[int64]string)
	groupOrganizationIDs := make(sdk.StringSlice, 0, len(groupOrganizations))
	for i := range groupOrganizations {
		mGrpOrgs[groupOrganizations[i].GroupID] = groupOrganizations[i].OrganizationID
		groupOrganizationIDs = append(groupOrganizationIDs, groupOrganizations[i].OrganizationID)
	}
	groupOrganizationIDs.Unique()

	organizations, err := organization.LoadOrganizationByIDs(ctx, db, groupOrganizationIDs)
	if err != nil {
		return nil, err
	}
	mapOrganization := make(map[string]sdk.Organization)
	for _, o := range organizations {
		mapOrganization[o.ID] = o
	}

	for _, row := range dbResultSet {
		gp := sdk.GroupPermission{
			Permission: row.Role,
			Group: sdk.Group{
				ID:   row.GroupID,
				Name: row.GroupName,
			},
		}
		if orgID, ok := mGrpOrgs[gp.Group.ID]; ok {
			gp.Group.Organization = mapOrganization[orgID].Name
		}
		result[row.WorkflowID] = append(result[row.WorkflowID], gp)
	}

	return result, nil
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

func CheckWorkflowGroups(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project, w *sdk.Workflow, consumer *sdk.AuthUserConsumer) error {
	if err := LoadGroupsIntoProject(ctx, db, proj); err != nil {
		return err
	}
	for i := range w.Groups {
		if err := CheckGroupPermission(ctx, db, proj.ProjectGroups, &w.Groups[i], consumer); err != nil {
			return err
		}
	}
	for _, n := range w.WorkflowData.Array() {
		for i := range n.Groups {
			if err := CheckGroupPermission(ctx, db, proj.ProjectGroups, &n.Groups[i], consumer); err != nil {
				return err
			}
		}
	}
	return nil
}
