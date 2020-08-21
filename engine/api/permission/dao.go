package permission

import (
	"context"
	"database/sql"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func LoadWorkflowMaxLevelPermissionByWorkflowIDs(ctx context.Context, db gorp.SqlExecutor, workflowIDs []int64, groupIDs []int64) (sdk.EntitiesPermissions, error) {
	_, end := telemetry.Span(ctx, "permission.LoadWorkflowMaxLevelPermissionByWorkflowIDs")
	defer end()

	query := `
		SELECT workflow.id::text, max(workflow_perm.role)
		FROM workflow_perm
		JOIN workflow ON workflow.id = workflow_perm.workflow_id
		JOIN project ON project.id = workflow.project_id
		JOIN project_group ON project_group.id = workflow_perm.project_group_id
		WHERE project_group.project_id = project.id
		AND workflow.id = ANY($1)
		AND project_group.group_id = ANY($2)
		GROUP BY workflow.id, workflow.name`

	rows, err := db.Query(query, pq.Int64Array(workflowIDs), pq.Int64Array(groupIDs))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	defer rows.Close()

	return scanPermissions(rows)
}

func LoadWorkflowMaxLevelPermission(ctx context.Context, db gorp.SqlExecutor, projectKey string, workflowNames []string, groupIDs []int64) (sdk.EntitiesPermissions, error) {
	_, end := telemetry.Span(ctx, "permission.LoadWorkflowMaxLevelPermission")
	defer end()

	query := `
		SELECT workflow.name, max(workflow_perm.role)
		FROM workflow_perm
		JOIN workflow ON workflow.id = workflow_perm.workflow_id
		JOIN project ON project.id = workflow.project_id
		JOIN project_group ON project_group.id = workflow_perm.project_group_id
		WHERE project_group.project_id = project.id
		AND project.projectkey = $1
		AND workflow.name = ANY(string_to_array($2, ','))
		AND project_group.group_id = ANY(string_to_array($3, ',')::int[])
		GROUP BY workflow.id, workflow.name`

	rows, err := db.Query(query, projectKey, strings.Join(workflowNames, ","), gorpmapping.IDsToQueryString(groupIDs))
	if err == sql.ErrNoRows {
		return sdk.EntitiesPermissions{}, nil
	}
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	defer rows.Close()

	return scanPermissions(rows)
}

func LoadProjectMaxLevelPermission(ctx context.Context, db gorp.SqlExecutor, projectKeys []string, groupIDs []int64) (sdk.EntitiesPermissions, error) {
	_, end := telemetry.Span(ctx, "permission.LoadProjectMaxLevelPermission")
	defer end()

	query := `
		SELECT project.projectkey, max(project_group.role)
		FROM project_group
		JOIN project ON project.id = project_group.project_id
		AND project.projectkey = ANY(string_to_array($1, ','))
		AND project_group.group_id = ANY(string_to_array($2, ',')::int[])
		GROUP BY project.projectkey`

	rows, err := db.Query(query, strings.Join(projectKeys, ","), gorpmapping.IDsToQueryString(groupIDs))
	if err == sql.ErrNoRows {
		return sdk.EntitiesPermissions{}, nil
	}
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	defer rows.Close()

	return scanPermissions(rows)
}

func scanPermissions(rows *sql.Rows) (sdk.EntitiesPermissions, error) {
	var res = sdk.EntitiesPermissions{}
	for rows.Next() {
		var k string
		var p sdk.Permissions
		var i int64
		if err := rows.Scan(&k, &i); err != nil {
			return nil, sdk.WithStack(err)
		}
		switch i {
		case sdk.PermissionRead:
			p.Readable = true
		case sdk.PermissionReadExecute:
			p.Readable = true
			p.Executable = true
		case sdk.PermissionReadWriteExecute:
			p.Readable = true
			p.Executable = true
			p.Writable = true
		}
		res[k] = p
	}
	return res, nil
}

// AccessToWorkflowNode check rights on the given workflow node
func AccessToWorkflowNode(ctx context.Context, db gorp.SqlExecutor, wf *sdk.Workflow, wn *sdk.Node, u sdk.AuthConsumer, access int) bool {
	if wn == nil {
		return false
	}

	if u.Admin() {
		return true
	}

	if len(wn.Groups) > 0 {
		for _, id := range u.GetGroupIDs() {
			if id == group.SharedInfraGroup.ID {
				return true
			}
			for _, grp := range wn.Groups {
				if id == grp.Group.ID && grp.Permission >= access {
					return true
				}
			}
		}
		return false
	}

	perms, _ := LoadWorkflowMaxLevelPermission(ctx, db, wf.ProjectKey, []string{wf.Name}, u.GetGroupIDs())
	return perms.Level(wf.Name) >= access
}
