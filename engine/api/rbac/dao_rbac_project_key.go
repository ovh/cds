package rbac

import (
	"context"
	"github.com/ovh/cds/sdk/telemetry"

	"github.com/lib/pq"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getAllRBACProjectKeys(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacProjectKey, error) {
	var rbacProjectIdentifier []rbacProjectKey
	if err := gorpmapping.GetAll(ctx, db, q, &rbacProjectIdentifier); err != nil {
		return nil, err
	}
	rbacProjectIdentifierFiltered := make([]rbacProjectKey, 0, len(rbacProjectIdentifier))
	for _, projectDatas := range rbacProjectIdentifier {
		isValid, err := gorpmapping.CheckSignature(projectDatas, projectDatas.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_project_keys %d", projectDatas.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACProjectKeys> rbac_project_keys %d data corrupted", projectDatas.ID)
			continue
		}
		rbacProjectIdentifierFiltered = append(rbacProjectIdentifierFiltered, projectDatas)
	}
	return rbacProjectIdentifierFiltered, nil
}

func loadRBACProjectKeys(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_project_keys WHERE rbac_project_id = $1").Args(rbacProject.ID)
	rbacProjectKeys, err := getAllRBACProjectKeys(ctx, db, q)
	if err != nil {
		return err
	}
	rbacProject.RBACProject.RBACProjectKeys = make([]string, 0, len(rbacProjectKeys))
	for _, projectDatas := range rbacProjectKeys {
		rbacProject.RBACProject.RBACProjectKeys = append(rbacProject.RBACProject.RBACProjectKeys, projectDatas.ProjectKey)
	}
	return nil
}

func loadAllRBACProjectKeys(ctx context.Context, db gorp.SqlExecutor, rbacProjectIDs []int64) ([]rbacProjectKey, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM rbac_project_keys WHERE rbac_project_id = ANY($1)`).Args(pq.Int64Array(rbacProjectIDs))
	return getAllRBACProjectKeys(ctx, db, query)
}

func HasRoleOnProjectAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, userID string, projectKey string) (bool, error) {
	ctx, next := telemetry.Span(ctx, "rbac.HasRoleOnProjectAndUserID")
	defer next()
	projectKeys, err := LoadProjectKeysByRoleAndUserID(ctx, db, role, userID)
	if err != nil {
		return false, err
	}
	return sdk.IsInArray(projectKey, projectKeys), nil
}

func LoadProjectKeysByRoleAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, userID string) ([]string, error) {
	// Get rbac_project_groups
	rbacProjectGroups, err := loadRBACProjectGroupsByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	// Get rbac_project_users
	rbacProjectUsers, err := loadRBACProjectUsersByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}

	// Deduplicate rbac_project.id
	rbacProjectIDs := make(sdk.Int64Slice, 0)
	for _, rpg := range rbacProjectGroups {
		rbacProjectIDs = append(rbacProjectIDs, rpg.RbacProjectID)
	}
	for _, rpu := range rbacProjectUsers {
		rbacProjectIDs = append(rbacProjectIDs, rpu.RbacProjectID)
	}
	rbacProjectIDs.Unique()

	// Get rbac_project
	rbacProjects, err := loadRBACProjectsByRoleAndIDs(ctx, db, role, rbacProjectIDs)
	if err != nil {
		return nil, err
	}

	// Get rbac_project_keys
	rbacProjectIDs = make([]int64, 0, len(rbacProjects))
	for _, rp := range rbacProjects {
		rbacProjectIDs = append(rbacProjectIDs, rp.ID)
	}
	rbacProjectKeys, err := loadAllRBACProjectKeys(ctx, db, rbacProjectIDs)
	if err != nil {
		return nil, err
	}

	// Deduplicate project keys
	projectKeys := make([]string, 0)
	projectMap := make(map[string]struct{})
	for _, rpi := range rbacProjectKeys {
		if _, has := projectMap[rpi.ProjectKey]; !has {
			projectMap[rpi.ProjectKey] = struct{}{}
			projectKeys = append(projectKeys, rpi.ProjectKey)
		}
	}
	return projectKeys, nil
}
