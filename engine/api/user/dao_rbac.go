package user

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func LoadUsersByRbacGlobal(ctx context.Context, db gorp.SqlExecutor, rbacGlobalID int64) ([]sdk.AuthentifiedUser, error) {
	query := `
		SELECT u.*
		FROM rbac_global_users rgu
		JOIN authentified_user u ON u.id = rgu.user_id
		WHERE rgu.rbac_global_id = $1
	`
	return getAll(ctx, db, gorpmapping.NewQuery(query).Args(rbacGlobalID))
}

func LoadUsersByRbacProject(ctx context.Context, db gorp.SqlExecutor, rbacProjectID int64) ([]sdk.AuthentifiedUser, error) {
	query := `
		SELECT u.*
		FROM rbac_project_users rpu
		JOIN authentified_user u ON u.id = rpu.user_id
		WHERE rpu.rbac_project_id = $1
	`
	return getAll(ctx, db, gorpmapping.NewQuery(query).Args(rbacProjectID))
}
