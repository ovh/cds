package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

func getAllRBACRegionProject(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacRegionProject, error) {
	var rbacRegionProjects []rbacRegionProject
	if err := gorpmapping.GetAll(ctx, db, q, &rbacRegionProjects); err != nil {
		return nil, err
	}
	rbacRegionProjectsFiltered := make([]rbacRegionProject, 0, len(rbacRegionProjects))
	for _, regionProject := range rbacRegionProjects {
		isValid, err := gorpmapping.CheckSignature(regionProject, regionProject.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_region_project %d", regionProject.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACRegionProjectKeys> rbac_region_project %d data corrupted", regionProject.ID)
			continue
		}
		rbacRegionProjectsFiltered = append(rbacRegionProjectsFiltered, regionProject)
	}
	return rbacRegionProjectsFiltered, nil
}

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

func HasRoleOnRegionProject(ctx context.Context, db gorp.SqlExecutor, role string, regionID string, projectKey string) (bool, error) {
	ctx, next := telemetry.Span(ctx, "rbac.HasRoleOnRegionProject")
	defer next()

	rbacRegionProjectKeyProject, err := loadRBACRegionProjectByProjectKey(ctx, db, projectKey)
	if err != nil {
		return false, err
	}

	rbacRegionProjectID := sdk.Int64Slice{}
	for _, rrp := range rbacRegionProjectKeyProject {
		rbacRegionProjectID = append(rbacRegionProjectID, rrp.RbacRegionProjectID)
	}
	rbacRegionProjectID.Unique()

	if len(rbacRegionProjectID) == 0 {
		return false, nil
	}

	rbacRegionProjects, err := loadRBACRegionProjectByIDs(ctx, db, role, regionID, rbacRegionProjectID)
	if err != nil {
		return false, err
	}

	return len(rbacRegionProjects) > 0, nil
}

func loadRBACRegionProjectByIDs(ctx context.Context, db gorp.SqlExecutor, role string, regionID string, IDs []int64) ([]rbacRegionProject, error) {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_region_project WHERE role = $1 AND region_id = $2 AND ID = ANY($3)").Args(role, regionID, pq.Int64Array(IDs))
	return getAllRBACRegionProject(ctx, db, q)
}
