package rbac

import (
	"context"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
)

func insertRBACRegionProject(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacRegionProject *rbacRegionProject) error {
	if err := gorpmapping.InsertAndSign(ctx, db, rbacRegionProject); err != nil {
		return err
	}

	for _, projectKey := range rbacRegionProject.RBACProjectKeys {
		if err := insertRBACRegionProjectKey(ctx, db, rbacRegionProject.ID, projectKey); err != nil {
			return err
		}
	}

	return nil
}
