package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func insertRBACRegionUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacRegionID int64, userID string) error {
	rgu := rbacRegionUser{
		RbacRegionID: rbacRegionID,
		RbacUserID:   userID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func loadRBACRegionUsersByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) ([]rbacRegionUser, error) {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_region_users WHERE user_id = $1").Args(userID)
	return getAllRBACRegionUsers(ctx, db, q)
}

func loadRBACRegionUsers(ctx context.Context, db gorp.SqlExecutor, rbacRegion *rbacRegion) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_region_users WHERE rbac_region_id = $1").Args(rbacRegion.ID)
	rbacUserIDS, err := getAllRBACRegionUsers(ctx, db, q)
	if err != nil {
		return err
	}
	rbacRegion.RBACRegion.RBACUsersIDs = make([]string, 0, len(rbacUserIDS))
	for _, rbacUsers := range rbacUserIDS {
		rbacRegion.RBACRegion.RBACUsersIDs = append(rbacRegion.RBACRegion.RBACUsersIDs, rbacUsers.RbacUserID)
	}
	return nil
}

func getAllRBACRegionUsers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacRegionUser, error) {
	var rbacRegionUsers []rbacRegionUser
	if err := gorpmapping.GetAll(ctx, db, q, &rbacRegionUsers); err != nil {
		return nil, err
	}

	usersFiltered := make([]rbacRegionUser, 0, len(rbacRegionUsers))
	for _, rbacUsers := range rbacRegionUsers {
		isValid, err := gorpmapping.CheckSignature(rbacUsers, rbacUsers.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_region_users %d", rbacUsers.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACRegionUsers> rbac_region_users %d data corrupted", rbacUsers.ID)
			continue
		}
		usersFiltered = append(usersFiltered, rbacUsers)
	}
	return usersFiltered, nil
}
