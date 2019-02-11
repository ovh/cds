package migrate

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type nodePerm struct {
	NodeID     int64 `db:"id"`
	WorkflowID int64 `db:"workflow_id"`
	GroupID    int64 `db:"group_id"`
	Role       int   `db:"role"`
}

type customGroupPermission struct {
	nodeID int64
	sdk.GroupPermission
}

// Permissions function to migrate all previous permissions
func Permissions(DBFunc func() *gorp.DbMap, store cache.Store) error {
	db := DBFunc()
	projects, err := project.LoadAll(context.Background(), DBFunc(), store, &sdk.User{Admin: true}, project.LoadOptions.WithPermission, project.LoadOptions.WithWorkflowNames)
	if err != nil {
		return sdk.WrapError(err, "Cannot load all projects")
	}

	var globalError error
	for _, proj := range projects {
		// Add all workflow perm in new table
		for _, wf := range proj.WorkflowNames {
			groups, err := loadPreviousWorkflowGroups(db, wf.ID)
			if err != nil {
				log.Error("Migrate.Permissions> cannot load workflow %s/%s groups : %v", proj.Key, wf.Name, err)
				continue
			}
			if err := group.UpsertAllWorkflowGroups(db, &sdk.Workflow{ProjectID: proj.ID, ID: wf.ID}, groups); err != nil {
				log.Error("Migrate.Permissions> cannot upsert workflow %s/%s groups : %v", proj.Key, wf.Name, err)
			}
		}

		// load all environments/pipelines/applications permission
		envsGr, errEnv := loadEnvironmentsPermission(db, proj.ID)
		if errEnv != nil {
			log.Error("%v", sdk.WrapError(errEnv, "cannot load environment permissions"))
			continue
		}
		appGr, errApp := loadApplicationsPermission(db, proj.ID)
		if errApp != nil {
			log.Error("%v", sdk.WrapError(errApp, "cannot load application permissions"))
			continue
		}
		pipGr, errPip := loadPipelinesPermission(db, proj.ID)
		if errPip != nil {
			log.Error("%v", sdk.WrapError(errPip, "cannot load pipeline permissions"))
			continue
		}

		permissions := mergePermissions(envsGr, appGr)
		permissions = mergePermissions(permissions, pipGr)
		permToAdd := diffPermission(proj.ProjectGroups, permissions)

		// Add missing permissions that was linked to apps/pips/envs
		for _, perm := range permToAdd {
			queryInsert := `INSERT INTO project_group (project_id, group_id, role) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`
			if _, err := db.Exec(queryInsert, proj.ID, perm.Group.ID, perm.Permission); err != nil {
				log.Error("%v", sdk.WrapError(err, "cannot insert group %s with id %d in project %s with id %d", perm.Group.Name, perm.Group.ID, proj.Key, proj.ID))
			}
		}

		// For each workflow node with environement linked add
		// Get all workflow node id and environment id
		//
		querySelect := `SELECT w_node.id, w_node.workflow_id, environment_group.group_id, environment_group.role
		FROM workflow
			JOIN w_node ON workflow.id = w_node.workflow_id
			JOIN w_node_context ON w_node_context.node_id = w_node.id
			JOIN environment_group ON environment_group.environment_id = w_node_context.environment_id 
		WHERE workflow.project_id = $1 AND environment_group.role > $2`
		var nodePerms []nodePerm
		if _, err := db.Select(&nodePerms, querySelect, proj.ID, permission.PermissionRead); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			log.Error("%v", sdk.WrapError(err, "cannot load node permissions for project %s with id %d", proj.Key, proj.ID))
			continue
		}

		gpsByWorkflow := map[int64][]customGroupPermission{}
		for _, nodeperm := range nodePerms {
			gpsByWorkflow[nodeperm.WorkflowID] = append(gpsByWorkflow[nodeperm.WorkflowID], customGroupPermission{
				GroupPermission: sdk.GroupPermission{
					Group:      sdk.Group{ID: nodeperm.GroupID},
					Permission: nodeperm.Role,
				},
				nodeID: nodeperm.NodeID,
			})
		}
		ctx := context.Background()
		for workflowID, gps := range gpsByWorkflow {
			projLoaded, errP := project.LoadByID(db, store, proj.ID, &sdk.User{Admin: true})
			if errP != nil {
				log.Error("%v", sdk.WrapError(errP, "cannot load project %s", proj.Key))
				continue
			}

			oldWf, err := workflow.LoadByID(db, store, projLoaded, workflowID, &sdk.User{Admin: true}, workflow.LoadOptions{})
			if err != nil {
				log.Error("%v", sdk.WrapError(err, "cannot load workflow id %d in project %s", workflowID, projLoaded.Key))
				continue
			}

			if oldWf.ToDelete {
				continue
			}
			log.Info("migrate.Permissions> Workflow %s", oldWf.Name)
			newWf := *oldWf

			added := 0
			for _, gp := range gps {
				node := newWf.WorkflowData.NodeByID(gp.nodeID)
				if node == nil {
					continue
				}
				found := false
				for _, nodeGr := range node.Groups {
					if nodeGr.Group.ID == gp.Group.ID {
						found = true
						break
					}
				}
				if !found {
					exist := false
					for _, grWf := range newWf.Groups {
						if grWf.Group.ID == gp.Group.ID {
							exist = true
							break
						}
					}
					if exist {
						added++
						node.Groups = append(node.Groups, gp.GroupPermission)
					}
				}
			}

			if added == 0 {
				continue
			}
			tx, errTx := db.Begin()
			if errTx != nil {
				log.Error("%v", sdk.WrapError(errTx, "cannot begin transaction"))
				continue
			}

			if err := workflow.Update(ctx, tx, store, &newWf, oldWf, projLoaded, &sdk.User{Admin: true}); err != nil {
				_ = tx.Rollback()
				if globalError == nil {
					globalError = err
				} else {
					globalError = sdk.WrapError(globalError, err.Error())
				}
				log.Warning("migrate.Permissions> cannot update workflow %s/%s : %v", projLoaded.Key, newWf.Name, err)
			} else {
				if err := tx.Commit(); err != nil {
					log.Error("%v", sdk.WrapError(err, "cannot commit transaction"))
				}
			}
		}
	}

	return globalError
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
		return nil, sdk.WithStack(err)
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
		return nil, sdk.WithStack(err)
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
		return nil, sdk.WithStack(errq)
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
