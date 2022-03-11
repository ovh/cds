package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func LoadGroupByRbacGlobal(ctx context.Context, db gorp.SqlExecutor, rbacGlobalID int64) ([]sdk.Group, error) {
	query := `
		SELECT g.*
		FROM rbac_global_groups rgg
		JOIN "group" g ON g.id = rgg.group_id
		WHERE rgg.rbac_global_id = $1
	`
	return getAll(ctx, db, gorpmapping.NewQuery(query).Args(rbacGlobalID))
}

func LoadGroupByRbacProject(ctx context.Context, db gorp.SqlExecutor, rbacProjectID int64) ([]sdk.Group, error) {
	query := `
		SELECT g.*
		FROM rbac_project_groups rpg
		JOIN "group" g ON g.id = rpg.group_id
		WHERE rpg.rbac_project_id = $1
	`
	return getAll(ctx, db, gorpmapping.NewQuery(query).Args(rbacProjectID))
}
