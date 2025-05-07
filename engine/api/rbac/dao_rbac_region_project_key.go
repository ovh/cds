package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func getAllRBACRegionProjectKeys(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacRegionProjectKey, error) {
	var rbacRegionProjectIdentifier []rbacRegionProjectKey
	if err := gorpmapping.GetAll(ctx, db, q, &rbacRegionProjectIdentifier); err != nil {
		return nil, err
	}
	rbacProjectIdentifierFiltered := make([]rbacRegionProjectKey, 0, len(rbacRegionProjectIdentifier))
	for _, projectDatas := range rbacRegionProjectIdentifier {
		isValid, err := gorpmapping.CheckSignature(projectDatas, projectDatas.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_region_project_keys_project %d", projectDatas.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACRegionProjectKeys> rbac_region_project_keys_project %d data corrupted", projectDatas.ID)
			continue
		}
		rbacProjectIdentifierFiltered = append(rbacProjectIdentifierFiltered, projectDatas)
	}
	return rbacProjectIdentifierFiltered, nil
}

func insertRBACRegionProjectKey(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacParentID int64, projectKey string) error {
	rpk := rbacRegionProjectKey{
		RbacRegionProjectID: rbacParentID,
		ProjectKey:          projectKey,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rpk); err != nil {
		return err
	}
	return nil
}

func loadRBACRegionProjectKeys(ctx context.Context, db gorp.SqlExecutor, rbacRegionProject *rbacRegionProject) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_region_project_keys_project WHERE rbac_region_project_id = $1").Args(rbacRegionProject.ID)
	rbacRegionProjectKeys, err := getAllRBACRegionProjectKeys(ctx, db, q)
	if err != nil {
		return err
	}
	rbacRegionProject.RBACRegionProject.RBACProjectKeys = make([]string, 0, len(rbacRegionProjectKeys))
	for _, projectDatas := range rbacRegionProjectKeys {
		rbacRegionProject.RBACRegionProject.RBACProjectKeys = append(rbacRegionProject.RBACRegionProject.RBACProjectKeys, projectDatas.ProjectKey)
	}
	return nil
}

func loadRBACRegionProjectByProjectKey(ctx context.Context, db gorp.SqlExecutor, key string) ([]rbacRegionProjectKey, error) {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_region_project_keys_project WHERE project_key = $1").Args(key)
	return getAllRBACRegionProjectKeys(ctx, db, q)
}
