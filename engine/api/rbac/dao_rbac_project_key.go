package rbac

import (
	"context"
	"encoding/json"

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

func HasRoleOnProjectAndVCSUser(ctx context.Context, db gorp.SqlExecutor, role string, user sdk.RBACVCSUser, projectKey string) (bool, error) {
	projectKeys, err := LoadAllProjectKeysAllowedForVCSUser(ctx, db, role, user)
	if err != nil {
		return false, err
	}
	return sdk.IsInArray(projectKey, projectKeys), nil
}

func HasRoleOnProjectAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, userID string, projectKey string) (bool, error) {
	ctx, next := telemetry.Span(ctx, "rbac.HasRoleOnProjectAndUserID")
	defer next()
	projectKeys, err := LoadAllProjectKeysAllowed(ctx, db, role, userID)
	if err != nil {
		return false, err
	}
	return sdk.IsInArray(projectKey, projectKeys), nil
}

func loadPublicProjectKeysByRole(ctx context.Context, db gorp.SqlExecutor, role string) (sdk.StringSlice, error) {
	rbacProjects, err := loadRBACProjectByRoleAndPublic(ctx, db, role)
	if err != nil {
		return nil, err
	}

	ids := make(sdk.Int64Slice, 0, len(rbacProjects))
	for _, rp := range rbacProjects {
		ids = append(ids, rp.ID)
	}
	ids.Unique()

	rbacProjectKeys, err := loadAllRBACProjectKeys(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	projectKeys := make(sdk.StringSlice, 0, len(rbacProjectKeys))
	for _, rpi := range rbacProjectKeys {
		projectKeys = append(projectKeys, rpi.ProjectKey)
	}
	projectKeys.Unique()
	return projectKeys, nil
}

func LoadAllProjectKeysAllowedForVCSUser(ctx context.Context, db gorp.SqlExecutor, role string, user sdk.RBACVCSUser) (sdk.StringSlice, error) {
	btes, _ := json.Marshal([]sdk.RBACVCSUser{user})
	var ids []int64
	_, err := db.Select(&ids, "select id from rbac_project where role = $1 and vcs_users::JSONB @> $2", role, string(btes))
	if err != nil {
		return nil, err
	}
	keys, err := loadAllRBACProjectKeys(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	projectKeys := make(sdk.StringSlice, 0, len(keys))
	for _, rpi := range keys {
		projectKeys = append(projectKeys, rpi.ProjectKey)
	}
	projectKeys.Unique()

	log.Debug(ctx, "LoadAllProjectKeysAllowedForVCSUser> %s has role %q on %+v", string(btes), role, projectKeys)

	return projectKeys, nil
}

func LoadAllProjectKeysAllowed(ctx context.Context, db gorp.SqlExecutor, role string, userID string) (sdk.StringSlice, error) {
	keysByUsers, err := loadProjectKeysByRoleAndUserID(ctx, db, role, userID)
	if err != nil {
		return nil, err
	}
	keysPublic, err := loadPublicProjectKeysByRole(ctx, db, role)
	if err != nil {
		return nil, err
	}
	keysByUsers = append(keysByUsers, keysPublic...)
	keysByUsers.Unique()
	return keysByUsers, nil
}

func loadProjectKeysByRoleAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, userID string) (sdk.StringSlice, error) {
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

	projectKeys := make(sdk.StringSlice, 0, len(rbacProjectKeys))
	for _, rpi := range rbacProjectKeys {
		projectKeys = append(projectKeys, rpi.ProjectKey)
	}
	projectKeys.Unique()
	return projectKeys, nil
}
