package project

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func LoadProjectByRbacProject(ctx context.Context, db gorp.SqlExecutor, rbacProjectID int64) ([]sdk.Project, error) {
	query := `
		SELECT p.*
		FROM rbac_project_ids rpi
		JOIN project p ON p.id = rpi.project_id
		WHERE rpi.rbac_project_id = $1`
	return loadprojects(ctx, db, nil, query, rbacProjectID)
}
