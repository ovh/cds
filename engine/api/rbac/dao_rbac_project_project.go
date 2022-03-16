package rbac

import (
	"context"
	"github.com/lib/pq"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getAllRBACProjectIdentifiers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacProjectIdentifiers, error) {
	var rbacProjectIdentifier []rbacProjectIdentifiers
	if err := gorpmapping.GetAll(ctx, db, q, &rbacProjectIdentifier); err != nil {
		return nil, err
	}
	for _, projectDatas := range rbacProjectIdentifier {
		isValid, err := gorpmapping.CheckSignature(projectDatas, projectDatas.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_project_projects %d", projectDatas.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACProjectIdentifiers> rbac_project_projects %d data corrupted", projectDatas.ID)
			continue
		}
	}
	return rbacProjectIdentifier, nil
}

func loadRBACProjectIdentifiers(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	q := gorpmapping.NewQuery("SELECT * FROM  rbac_project_projects WHERE rbac_project_id = $1").Args(rbacProject.ID)
	rbacProjectIdentifiers, err := getAllRBACProjectIdentifiers(ctx, db, q)
	if err != nil {
		return err
	}
	rbacProject.RBACProject.RBACProjectsIDs = make([]int64, 0, len(rbacProjectIdentifiers))
	for _, projectDatas := range rbacProjectIdentifiers {
		rbacProject.RBACProject.RBACProjectsIDs = append(rbacProject.RBACProject.RBACProjectsIDs, projectDatas.ProjectID)
	}
	return nil
}

func loadRRBACProjectIdentifiers(ctx context.Context, db gorp.SqlExecutor, rbacProjectIDs []int64) ([]rbacProjectIdentifiers, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM rbac_project_projects WHERE rbac_project_id = ANY($1)`).Args(pq.Int64Array(rbacProjectIDs))
	return getAllRBACProjectIdentifiers(ctx, db, query)
}

func LoadProjectIDsByRoleAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, userID string) ([]int64, error) {
	// Get rbac_project_groups
	rbacProjectGroups, err := loadRbacProjectGroupsByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	// Get rbac_project_users
	rbacProjectUsers, err := loadRbacProjectUsersByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}

	// Deduplicate rbac_project.id
	mapRbacProjectID := make(map[int64]struct{})
	rbacProjectIDs := make([]int64, 0)
	for _, rpg := range rbacProjectGroups {
		mapRbacProjectID[rpg.RbacProjectID] = struct{}{}
		rbacProjectIDs = append(rbacProjectIDs, rpg.RbacProjectID)
	}
	for _, rpu := range rbacProjectUsers {
		if _, has := mapRbacProjectID[rpu.RbacProjectID]; !has {
			mapRbacProjectID[rpu.RbacProjectID] = struct{}{}
			rbacProjectIDs = append(rbacProjectIDs, rpu.RbacProjectID)
		}
	}

	// Get rbac_project
	rbacProjects, err := loadRbacProjectsByRoleAndIDs(ctx, db, role, rbacProjectIDs)
	if err != nil {
		return nil, err
	}

	// Get rbac_project_projects
	rbacProjectIDs = make([]int64, 0, len(rbacProjects))
	for _, rp := range rbacProjects {
		rbacProjectIDs = append(rbacProjectIDs, rp.ID)
	}
	rbacProjectsIdentifiers, err := loadRRBACProjectIdentifiers(ctx, db, rbacProjectIDs)
	if err != nil {
		return nil, err
	}

	// Deduplicate project ID
	projectIDs := make([]int64, 0)
	projectMap := make(map[int64]struct{})
	for _, rpi := range rbacProjectsIdentifiers {
		if _, has := projectMap[rpi.ProjectID]; !has {
			projectMap[rpi.ProjectID] = struct{}{}
			projectIDs = append(projectIDs, rpi.ProjectID)
		}
	}
	return projectIDs, nil
}
