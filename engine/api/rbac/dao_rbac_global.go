package rbac

import (
	"context"
	"github.com/lib/pq"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func insertRBACGlobal(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rg *rbacGlobal) error {
	if err := gorpmapping.InsertAndSign(ctx, db, rg); err != nil {
		return err
	}

	for _, userID := range rg.RBACUsersIDs {
		if err := insertRBACGlobalUser(ctx, db, rg.ID, userID); err != nil {
			return err
		}
	}
	for _, groupID := range rg.RBACGroupsIDs {
		if err := insertRBACGlobalGroup(ctx, db, rg.ID, groupID); err != nil {
			return err
		}
	}
	return nil
}

func insertRBACGlobalUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacGlobalID int64, userID string) error {
	rgu := rbacGlobalUser{
		RbacGlobalID:     rbacGlobalID,
		RbacGlobalUserID: userID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func insertRBACGlobalGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacGlobalID int64, groupID int64) error {
	rgu := rbacGlobalGroup{
		RbacGlobalID:      rbacGlobalID,
		RbacGlobalGroupID: groupID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func HasGlobalRole(ctx context.Context, db gorp.SqlExecutor, role string, userID string) (bool, error) {
	ctx, next := telemetry.Span(ctx, "rbac.HasGlobalRole")
	defer next()

	// Get rbac_global_groups
	rbacGlobalGroups, err := loadRBACGlobalGroupsByUserID(ctx, db, userID)
	if err != nil {
		return false, err
	}
	// Get rbac_global_users
	rbacGlobalUsers, err := loadRBACGlobalUsersByUserID(ctx, db, userID)
	if err != nil {
		return false, err
	}

	rbacGlobalIDs := make(sdk.Int64Slice, 0)
	for _, rgg := range rbacGlobalGroups {
		rbacGlobalIDs = append(rbacGlobalIDs, rgg.RbacGlobalID)
	}
	for _, rgu := range rbacGlobalUsers {
		rbacGlobalIDs = append(rbacGlobalIDs, rgu.RbacGlobalID)
	}
	rbacGlobalIDs.Unique()

	rgs, err := loadRBACGlobalByRoleAndIDs(ctx, db, role, rbacGlobalIDs)
	if err != nil {
		return false, err
	}

	return len(rgs) > 0, nil
}

func loadRBACGlobalByRoleAndIDs(ctx context.Context, db gorp.SqlExecutor, role string, rbacGlobalIDs []int64) ([]rbacGlobal, error) {
	q := gorpmapping.NewQuery(`SELECT * from rbac_global WHERE role = $1 AND id = ANY($2)`).Args(role, pq.Int64Array(rbacGlobalIDs))
	return getAllRBACGlobal(ctx, db, q)
}

func getAllRBACGlobal(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacGlobal, error) {
	var rbacGlobal []rbacGlobal
	if err := gorpmapping.GetAll(ctx, db, q, &rbacGlobal); err != nil {
		return nil, err
	}

	for _, rg := range rbacGlobal {
		isValid, err := gorpmapping.CheckSignature(rg, rg.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_global %d", rg.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACGlobal> rbac_global %d data corrupted", rg.ID)
			continue
		}
		rbacGlobal = append(rbacGlobal, rg)
	}
	return rbacGlobal, nil
}
