package notification

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

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

	//TODO: refactor this ugly scan
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
