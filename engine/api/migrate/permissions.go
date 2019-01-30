package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/permission"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk/log"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

type nodePerm struct {
	NodeID  int64 `db:"id"`
	GroupID int64 `db:"group_id"`
	Role    int   `db:"role"`
}

// Permissions function to migrate all previous permissions
func Permissions(DBFunc func() *gorp.DbMap, store cache.Store) error {
	db := DBFunc()
	projects, err := project.LoadAll(context.Background(), DBFunc(), store, &sdk.User{Admin: true}, project.LoadOptions.WithPermission, project.LoadOptions.WithWorkflowNames)
	if err != nil {
		return sdk.WrapError(err, "Cannot load all projects")
	}

	for _, proj := range projects {
		// Add all workflow perm in new table
		for _, wf := range proj.WorkflowNames {
			groups, err := loadPreviousWorkflowGroups(db, wf.ID)
			if err != nil {
				log.Error("Migrate.Permissions> cannot load workflow %s/%s groups", proj.Key, wf.Name)
				continue
			}
			if err := group.UpsertAllWorkflowGroups(db, &sdk.Workflow{ProjectID: proj.ID, ID: wf.ID}, groups); err != nil {
				log.Error("Migrate.Permissions> cannot upsert workflow %s/%s groups", proj.Key, wf.Name)
			}
		}

		// load all environments/pipelines/applications permission
		envsGr, errEnv := loadEnvironmentsPermission(db, proj.ID)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "cannot load environment permissions")
		}
		appGr, errApp := loadApplicationsPermission(db, proj.ID)
		if errApp != nil {
			return sdk.WrapError(errApp, "cannot load application permissions")
		}
		pipGr, errPip := loadPipelinesPermission(db, proj.ID)
		if errPip != nil {
			return sdk.WrapError(errPip, "cannot load pipeline permissions")
		}

		permissions := mergePermissions(envsGr, appGr)
		permissions = mergePermissions(permissions, pipGr)
		permToAdd := diffPermission(proj.ProjectGroups, permissions)

		// Add missing permissions that was linked to apps/pips/envs
		for _, perm := range permToAdd {
			queryInsert := `INSERT INTO project_group (project_id, group_id, role) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`
			if _, err := db.Exec(queryInsert, proj.ID, perm.Group.ID, perm.Permission); err != nil {
				return sdk.WrapError(err, "cannot insert group %s with id %d in project %s with id %d", perm.Group.Name, perm.Group.ID, proj.Key, proj.ID)
			}
		}

		// For each workflow node with environement linked add
		// Get all workflow node id and environment id
		//
		querySelect := `SELECT w_node.id, environment_group.group_id, environment_group.role
		FROM workflow
			JOIN w_node ON workflow.id = w_node.workflow_id
			JOIN w_node_context ON w_node_context.node_id = w_node.id
			JOIN environment_group ON environment_group.environment_id = w_node_context.environment_id 
		WHERE workflow.project_id = $1 AND environment_group.role > $2`
		var nodePerms []nodePerm
		if _, err := db.Select(&nodePerms, querySelect, proj.ID, permission.PermissionRead); err != nil {
			return sdk.WrapError(err, "cannot load node permissions for project %s with id %d", proj.Key, proj.ID)
		}

		for _, nodeperm := range nodePerms {
			gp := []sdk.GroupPermission{{
				Group:      sdk.Group{ID: nodeperm.GroupID},
				Permission: nodeperm.Role,
			}}
			if err := group.InsertGroupsInNode(db, gp, nodeperm.NodeID); err != nil {
				if errPG, ok := sdk.Cause(err).(*pq.Error); ok && errPG.Code == gorpmapping.ViolateUniqueKeyPGCode {
					continue
				}
				errConv, ok := sdk.Cause(err).(sdk.Error)
				fmt.Println("ici === ", errConv, ok)
				fmt.Println(">>>>", sdk.Cause(err))
				if sdk.ErrorIs(err, sdk.ErrGroupNotFoundInWorkflow) {
					continue
				}
				return sdk.WrapError(err, "cannot insert group %d in node %d", nodeperm.GroupID, nodeperm.NodeID)
			}
		}

	}

	return nil
}

func loadEnvironmentsPermission(db gorp.SqlExecutor, projectID int64) ([]sdk.GroupPermission, error) {
	query := `SELECT DISTINCT "group".id, "group".name, environment_group.role FROM "group"
				JOIN environment_group ON environment_group.group_id = "group".id
			   	JOIN environment ON environment_group.environment_id = environment.id
			WHERE environment.project_id = $1`

	rows, err := db.Query(query, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	var groups []sdk.GroupPermission
	for rows.Next() {
		var group sdk.Group
		var perm int
		if err := rows.Scan(&group.ID, &group.Name, &perm); err != nil {
			return groups, err
		}
		groups = append(groups, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return groups, nil
}

func loadPipelinesPermission(db gorp.SqlExecutor, projectID int64) ([]sdk.GroupPermission, error) {
	query := `SELECT DISTINCT "group".id, "group".name, pipeline_group.role FROM "group"
				JOIN pipeline_group ON pipeline_group.group_id = "group".id
			   	JOIN pipeline ON pipeline_group.pipeline_id = pipeline.id
			WHERE pipeline.project_id = $1`

	rows, err := db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []sdk.GroupPermission
	for rows.Next() {
		var group sdk.Group
		var perm int
		if err := rows.Scan(&group.ID, &group.Name, &perm); err != nil {
			return groups, err
		}
		groups = append(groups, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return groups, nil
}

func loadApplicationsPermission(db gorp.SqlExecutor, projectID int64) ([]sdk.GroupPermission, error) {
	query := `SELECT DISTINCT "group".id, "group".name, application_group.role FROM "group"
				JOIN application_group ON application_group.group_id = "group".id
			   	JOIN application ON application_group.application_id = application.id
			WHERE application.project_id = $1`

	rows, err := db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []sdk.GroupPermission
	for rows.Next() {
		var group sdk.Group
		var perm int
		if err := rows.Scan(&group.ID, &group.Name, &perm); err != nil {
			return groups, err
		}
		groups = append(groups, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return groups, nil
}

func mergePermissions(gps1, gps2 []sdk.GroupPermission) []sdk.GroupPermission {
	gpsMerged := make([]sdk.GroupPermission, 0, len(gps1))
	permMap := make(map[int64]int, len(gps1))
	for _, gp1 := range gps1 {
		for _, gp2 := range gps2 {
			if gp1.Group.ID == gp2.Group.ID {
				if gp1.Permission > gp2.Permission {
					gpsMerged = append(gpsMerged, gp1)
					permMap[gp1.Group.ID] = gp1.Permission
				} else {
					gpsMerged = append(gpsMerged, gp2)
					permMap[gp1.Group.ID] = gp2.Permission
				}
				break
			}
		}
		if _, ok := permMap[gp1.Group.ID]; !ok {
			gpsMerged = append(gpsMerged, gp1)
			permMap[gp1.Group.ID] = gp1.Permission
		}
	}

	for _, gp2 := range gps2 {
		if _, ok := permMap[gp2.Group.ID]; !ok {
			gpsMerged = append(gpsMerged, gp2)
			permMap[gp2.Group.ID] = gp2.Permission
		}
	}

	return gpsMerged
}

func diffPermission(initialGp, gpToMerge []sdk.GroupPermission) []sdk.GroupPermission {
	var gpToAdd []sdk.GroupPermission
	for _, gpMerge := range gpToMerge {
		found := false
		for _, initGp := range initialGp {
			if gpMerge.Group.ID == initGp.Group.ID {
				found = true
				break
			}
		}
		if !found {
			gpToAdd = append(gpToAdd, gpMerge)
		}
	}

	return gpToAdd
}

func loadPreviousWorkflowGroups(db gorp.SqlExecutor, workflowID int64) ([]sdk.GroupPermission, error) {
	wgs := []sdk.GroupPermission{}

	query := `SELECT "group".id, "group".name, workflow_group.role FROM "group"
	 		  JOIN workflow_group ON workflow_group.group_id = "group".id
	 		  WHERE workflow_group.workflow_id = $1 ORDER BY "group".name ASC`
	rows, errq := db.Query(query, workflowID)
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
