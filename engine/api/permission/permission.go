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
func ApplicationPermission(key string, appName string, u *sdk.User) int {
	if u.Admin {
		return PermissionReadWriteExecute
	}

	return u.Permissions.ApplicationsPerm[sdk.UserPermissionKey{Key: key, Name: appName}]
}

// ProjectPermission  Get the permission for the given project
func ProjectPermission(projectKey string, u *sdk.User) int {
	if u.Admin || u == nil {
		return PermissionReadWriteExecute
	}

	return u.Permissions.ProjectsPerm[projectKey]
}

// WorkflowPermission  Get the permission for the given workflow
func WorkflowPermission(key string, name string, u *sdk.User) int {
	if u.Admin {
		return PermissionReadWriteExecute
	}

	return u.Permissions.WorkflowsPerm[sdk.UserPermissionKey{Key: key, Name: name}]
}

// PipelinePermission  Get the permission for the given pipeline
func PipelinePermission(key string, name string, u *sdk.User) int {
	if u.Admin {
		return PermissionReadWriteExecute
	}

	return u.Permissions.PipelinesPerm[sdk.UserPermissionKey{Key: key, Name: name}]
}

// EnvironmentPermission  Get the permission for the given environment
func EnvironmentPermission(key string, name string, u *sdk.User) int {
	if u.Admin {
		return PermissionReadWriteExecute
	}

	return u.Permissions.EnvironmentsPerm[sdk.UserPermissionKey{Key: key, Name: name}]
}

// AccessToApplication check if we can modify the given application
func AccessToApplication(key string, name string, u *sdk.User, access int) bool {
	if u.Admin {
		return true
	}

	return u.Permissions.ApplicationsPerm[sdk.UserPermissionKey{Key: key, Name: name}] >= access
}

// AccessToPipeline check if we can modify the given pipeline
func AccessToPipeline(key string, env, pip string, u *sdk.User, access int) bool {
	if u.Admin {
		return true
	}

	for _, g := range u.Groups {
		if g.ID == SharedInfraGroupID {
			return true
		}
	}

	if u.Permissions.PipelinesPerm[sdk.UserPermissionKey{Key: key, Name: pip}] >= access {
		if env != sdk.DefaultEnv.Name {
			return AccessToEnvironment(key, env, u, access)
		}
		return true
	}

	return false
}

// AccessToEnvironment check if we can modify the given environment
func AccessToEnvironment(key, env string, u *sdk.User, access int) bool {
	if env != sdk.DefaultEnv.Name {
		return true
	}

	if u.Admin {
		return true
	}

	for _, g := range u.Groups {
		if g.ID == SharedInfraGroupID {
			return true
		}
	}

	return u.Permissions.EnvironmentsPerm[sdk.UserPermissionKey{Key: key, Name: env}] >= access
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
