package permission

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	// PermissionRead  read permission on the resource
	PermissionRead = 4
	// PermissionReadExecute  read & execute permission on the resource
	PermissionReadExecute = 5
	// PermissionReadWriteExecute read/execute/write permission on the resource
	PermissionReadWriteExecute = 7
)

var (
	// SharedInfraGroupID must be init from elsewhere with group.SharedInfraGroup
	SharedInfraGroupID int64
)

// ApplicationPermission  Get the permission for the given application
func ApplicationPermission(applicationID int64, user *sdk.User) int {
	if user.Admin {
		return PermissionReadWriteExecute
	}
	max := 0
	for _, g := range user.Groups {
		for _, ag := range g.ApplicationGroups {
			if ag.Application.ID == applicationID && ag.Permission > max {
				max = ag.Permission
			}
		}
	}
	return max
}

// ProjectPermission  Get the permission for the given project
func ProjectPermission(projectKey string, user *sdk.User) int {
	if user.Admin || user == nil {
		return PermissionReadWriteExecute
	}
	max := 0
	for _, g := range user.Groups {
		for _, pg := range g.ProjectGroups {
			if pg.Project.Key == projectKey && pg.Permission > max {
				max = pg.Permission
			}
		}
	}
	return max
}

// WorkflowPermission  Get the permission for the given workflow
func WorkflowPermission(workflowID int64, user *sdk.User) int {
	if user.Admin {
		return PermissionReadWriteExecute
	}
	max := 0
	for _, g := range user.Groups {
		for _, wg := range g.WorkflowGroups {
			if wg.Workflow.ID == workflowID && wg.Permission > max {
				max = wg.Permission
			}
		}
	}
	return max
}

// PipelinePermission  Get the permission for the given pipeline
func PipelinePermission(pipelineID int64, user *sdk.User) int {
	if user.Admin {
		return PermissionReadWriteExecute
	}
	max := 0
	for _, g := range user.Groups {
		for _, pg := range g.PipelineGroups {
			if pg.Pipeline.ID == pipelineID && pg.Permission > max {
				max = pg.Permission
			}
		}
	}
	return max
}

// EnvironmentPermission  Get the permission for the given environment
func EnvironmentPermission(envID int64, user *sdk.User) int {
	if user.Admin {
		return PermissionReadWriteExecute
	}
	max := 0
	for _, g := range user.Groups {
		for _, eg := range g.EnvironmentGroups {
			if eg.Environment.ID == envID && eg.Permission > max {
				max = eg.Permission
			}
		}
	}
	return max
}

// AccessToApplication check if we can modify the given application
func AccessToApplication(applicationID int64, user *sdk.User, access int) bool {
	if user.Admin {
		return true
	}

	for _, g := range user.Groups {
		if g.ID == SharedInfraGroupID {
			return true
		}
		for _, ag := range g.ApplicationGroups {
			if ag.Application.ID == applicationID && ag.Permission >= access {
				return true
			}
		}
	}
	return false
}

// AccessToPipeline check if we can modify the given pipeline
func AccessToPipeline(environmentID, pipelineID int64, user *sdk.User, access int) bool {
	if user.Admin {
		return true
	}

	for _, g := range user.Groups {
		if g.ID == SharedInfraGroupID {
			return true
		}
		for _, pg := range g.PipelineGroups {
			if pg.Pipeline.ID == pipelineID && pg.Permission >= access {
				if environmentID != sdk.DefaultEnv.ID {
					return AccessToEnvironment(environmentID, user, access)
				}
				return true
			}
		}
	}
	return false
}

// AccessToEnvironment check if we can modify the given environment
func AccessToEnvironment(envID int64, user *sdk.User, access int) bool {
	if envID == sdk.DefaultEnv.ID {
		return true
	}

	if user.Admin {
		return true
	}

	for _, g := range user.Groups {
		if g.ID == SharedInfraGroupID {
			return true
		}
		for _, eg := range g.EnvironmentGroups {
			if eg.Environment.ID == envID && eg.Permission >= access {
				return true
			}
		}
	}
	return false
}

// ApplicationPipelineEnvironmentUsers returns users list with expected access to application/pipeline/environment
func ApplicationPipelineEnvironmentUsers(db gorp.SqlExecutor, appID, pipID, envID int64, access int) ([]sdk.User, error) {
	var query string
	var args []interface{}

	if envID == sdk.DefaultEnv.ID {
		query = `
			SELECT 	DISTINCT "user".id, "user".username, "user".data
			FROM 	"group"
			JOIN 	application_group ON "group".id = application_group.group_id
			JOIN 	pipeline_group ON "group".id = pipeline_group.group_id
			JOIN	group_user ON "group".id = group_user.group_id
			JOIN 	"user" ON group_user.user_id = "user".id
			WHERE	application_group.application_id = $1
			AND	pipeline_group.pipeline_id = $2
			AND  	application_group.role >= $3
			AND  	pipeline_group.role >= $3
		`
		args = []interface{}{appID, pipID, access}
	} else {
		query = `
			SELECT 	DISTINCT "user".id, "user".username, "user".data
			FROM 	"group"
			JOIN 	application_group ON "group".id = application_group.group_id
			JOIN 	pipeline_group ON "group".id = pipeline_group.group_id
			JOIN 	environment_group ON "group".id = environment_group.group_id
			JOIN	group_user ON "group".id = group_user.group_id
			JOIN 	"user" ON group_user.user_id = "user".id
			WHERE	application_group.application_id = $1
			AND	pipeline_group.pipeline_id = $2
			AND 	environment_group.environment_id = $3
			AND  	application_group.role >= $4
			AND  	pipeline_group.role >= $4
			AND 	environment_group.role >= $4
		`
		args = []interface{}{appID, pipID, envID, access}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return []sdk.User{}, nil
		}
		return []sdk.User{}, err
	}
	defer rows.Close()

	users := []sdk.User{}
	for rows.Next() {
		u := sdk.User{}
		var data string
		if err := rows.Scan(&u.ID, &u.Username, &data); err != nil {
			log.Warning("permission.ApplicationPipelineEnvironmentGroups> error while scanning user : %s", err)
			continue
		}

		uTemp := &sdk.User{}
		if err := json.Unmarshal([]byte(data), uTemp); err != nil {
			log.Warning("permission.ApplicationPipelineEnvironmentGroups> error while parsing user : %s", err)
			continue
		}
		users = append(users, *uTemp)
	}
	return users, nil
}
