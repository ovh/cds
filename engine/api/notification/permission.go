package notification

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// applicationPipelineEnvironmentUsers returns users list with expected access to application/pipeline/environment
func applicationPipelineEnvironmentUsers(db gorp.SqlExecutor, appID, pipID, envID int64, access int) ([]sdk.User, error) {
	var query string
	var args []interface{}
	users := []sdk.User{}
	if envID == sdk.DefaultEnv.ID {
		query = `
			SELECT 	DISTINCT "user".id, "user".username, "user".data
			FROM 	"group"
			JOIN 	application_group ON "group".id = application_group.group_id
			JOIN 	pipeline_group ON "group".id = pipeline_group.group_id
			JOIN	group_user ON "group".id = group_user.group_id
			JOIN 	"user" ON group_user.user_id = "user".id
			WHERE	application_group.application_id = $1
			AND		pipeline_group.pipeline_id = $2
			AND  	application_group.role >= $3
			AND  	pipeline_group.role >= $3
			AND		"group".id <> $4
		`
		args = []interface{}{appID, pipID, access, permission.DefaultGroupID}
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
			AND		pipeline_group.pipeline_id = $2
			AND 	environment_group.environment_id = $3
			AND  	application_group.role >= $4
			AND  	pipeline_group.role >= $4
			AND 	environment_group.role >= $4
			AND		"group".id <> $5
		`
		args = []interface{}{appID, pipID, envID, access, permission.DefaultGroupID}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return users, nil
		}
		return users, err
	}
	defer rows.Close()

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

// projectPermissionUsers Get users that access to given project, without default group
func projectPermissionUsers(db gorp.SqlExecutor, projectID int64, access int) ([]sdk.User, error) {
	var query string
	users := []sdk.User{}
	query = `
			SELECT DISTINCT "user".id, "user".username, "user".data
			FROM "group"
			JOIN project_group ON "group".id = project_group.group_id
			JOIN group_user ON "group".id = group_user.group_id
	        JOIN "user" ON group_user.user_id = "user".id
			WHERE project_group.project_id = $1
			AND project_group.role >=$2
			AND	"group".id <> $3
		`

	rows, err := db.Query(query, projectID, access, permission.DefaultGroupID)
	if err != nil {
		if err == sql.ErrNoRows {
			return users, nil
		}
		return users, err
	}
	defer rows.Close()

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
