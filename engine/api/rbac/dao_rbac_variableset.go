package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/glob"
	"github.com/ovh/cds/sdk/telemetry"
)

func insertRBACVariableSet(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacVS *rbacVariableSet) error {
	if err := gorpmapping.InsertAndSign(ctx, db, rbacVS); err != nil {
		return err
	}

	for _, rbUserID := range rbacVS.RBACUsersIDs {
		if err := insertRBACVariableSetUser(ctx, db, rbacVS.ID, rbUserID); err != nil {
			return err
		}
	}
	for _, rbGroupID := range rbacVS.RBACGroupsIDs {
		if err := insertRBACVariableSetGroup(ctx, db, rbacVS.ID, rbGroupID); err != nil {
			return err
		}
	}
	return nil
}

func insertRBACVariableSetUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacVSID int64, userID string) error {
	rgu := rbacVariableSetUser{
		RbacVariableSetID:     rbacVSID,
		RbacVariableSetUserID: userID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func insertRBACVariableSetGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacVSID int64, groupID int64) error {
	rgu := rbacVariableSetGroup{
		RbacVariableSetID:      rbacVSID,
		RbacVariableSetGroupID: groupID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func getAllRBACVariableSets(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacVariableSet, error) {
	var rbacVariableSets []rbacVariableSet
	if err := gorpmapping.GetAll(ctx, db, q, &rbacVariableSets); err != nil {
		return nil, err
	}

	vsFiltered := make([]rbacVariableSet, 0, len(rbacVariableSets))
	for _, wDatas := range rbacVariableSets {
		isValid, err := gorpmapping.CheckSignature(wDatas, wDatas.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_variableset %d", wDatas.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACVariableSets> rbac_variableset %d data corrupted", wDatas.ID)
			continue
		}
		vsFiltered = append(vsFiltered, wDatas)
	}
	return vsFiltered, nil
}

func HasRoleOnVariableSetAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, userID string, projectKey string, vsName string) (bool, error) {
	ctx, next := telemetry.Span(ctx, "rbac.HasRoleOnVariableSetAndUserID")
	defer next()

	variableSets, allVariablesetAllowed, err := LoadAllVariableSetsAllowed(ctx, db, role, projectKey, userID)
	if err != nil {
		return false, err
	}
	if allVariablesetAllowed {
		return true, nil
	}
	for _, item := range variableSets {
		g := glob.New(item)
		r, err := g.MatchString(vsName)
		if err != nil {
			return false, err
		}
		if r != nil {
			return true, nil
		}
	}
	return false, nil
}

func HasRoleOnVariableSetsAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, userID string, projectKey string, vsNames []string) (bool, string, error) {
	ctx, next := telemetry.Span(ctx, "rbac.HasRoleOnVariableSetAndUserID")
	defer next()

	variableSets, allVariablesetAllowed, err := LoadAllVariableSetsAllowed(ctx, db, role, projectKey, userID)
	if err != nil {
		return false, "", err
	}
	if allVariablesetAllowed {
		return true, "", nil
	}
	for _, v := range vsNames {
		if !variableSets.Contains(v) {
			return false, v, nil
		}
	}
	return true, "", nil
}

func LoadAllVariableSetsAllowed(ctx context.Context, db gorp.SqlExecutor, role string, projectKey string, userID string) (sdk.StringSlice, bool, error) {
	variableSets := sdk.StringSlice{}

	groups, err := group.LoadAllByUserID(ctx, db, userID)
	if err != nil {
		return nil, false, err
	}
	groupIDs := make(sdk.Int64Slice, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}

	rbaVSs, err := loadRBACVariableSetsByProjectAndRole(ctx, db, projectKey, role)
	if err != nil {
		return nil, false, err
	}

	for _, rw := range rbaVSs {
		if rw.AllUsers {
			if rw.AllVariableSets {
				return nil, true, nil
			}
			variableSets = append(variableSets, rw.RBACVariableSetNames...)
			continue
		}
		for _, rbacUserID := range rw.RBACUsersIDs {
			if rbacUserID == userID {
				if rw.AllVariableSets {
					return nil, true, nil
				}
				variableSets = append(variableSets, rw.RBACVariableSetNames...)
				continue
			}
		}
		for _, groupID := range rw.RBACGroupsIDs {
			if groupIDs.Contains(groupID) {
				if rw.AllVariableSets {
					return nil, true, nil
				}
				variableSets = append(variableSets, rw.RBACVariableSetNames...)
				continue
			}
		}
	}
	return variableSets, false, nil
}

func loadRBACVariableSetsByProjectAndRole(ctx context.Context, db gorp.SqlExecutor, projectKey string, role string) ([]rbacVariableSet, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM rbac_variableset WHERE project_key = $1 AND role = $2`).Args(projectKey, role)
	rbacVss, err := getAllRBACVariableSets(ctx, db, query)
	if err != nil {
		return nil, err
	}
	for i := range rbacVss {
		rw := &rbacVss[i]
		if !rw.AllUsers {
			if err := loadRBACVariableSetGroups(ctx, db, rw); err != nil {
				return nil, err
			}
			if err := loadRBACVariableSetUsers(ctx, db, rw); err != nil {
				return nil, err
			}
		}
	}
	return rbacVss, nil
}
