package notification

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
)

// projectPermissionUsers Get users that access to given project, without default group
func projectPermissionUserIDs(ctx context.Context, db gorp.SqlExecutor, projectID int64, access int) ([]string, error) {
	// TODO refactor: we have to load the group_authentified_user entoty to ensure signature validation
	query := `
			SELECT DISTINCT "group_authentified_user".id
			FROM "group"
			JOIN project_group ON "group".id = project_group.group_id
			JOIN group_authentified_user ON "group".id = group_authentified_user.group_id
			WHERE project_group.project_id = $1
			AND project_group.role >=$2
			AND	"group".id <> $3
		`
	var defaultGroupID int64
	if group.DefaultGroup != nil {
		defaultGroupID = group.DefaultGroup.ID
	}

	q := gorpmapping.NewQuery(query).Args(projectID, access, defaultGroupID)

	var userIDs []string
	if err := gorpmapping.GetAll(ctx, db, q, &userIDs); err != nil {
		return nil, err
	}

	return userIDs, nil
}
