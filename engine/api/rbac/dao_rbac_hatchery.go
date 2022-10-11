package rbac

import (
	"context"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
)

func insertRBACHatchery(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacHatchery *rbacHatchery) error {
	if err := gorpmapping.InsertAndSign(ctx, db, rbacHatchery); err != nil {
		return err
	}
	return nil
}
