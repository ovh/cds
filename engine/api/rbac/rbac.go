package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func FillWithIDs(ctx context.Context, db gorp.SqlExecutor, r *sdk.RBAC) error {
	// Check existing permission
	rbacDB, err := LoadRBACByName(ctx, db, r.Name)
	if err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
	}
	if rbacDB != nil {
		r.ID = rbacDB.ID
	}

	userCache := make(map[string]string)
	groupCache := make(map[string]int64)
	for gID := range r.Globals {
		rbacGbl := &r.Globals[gID]
		if err := fillRBACGlobalWithID(ctx, db, rbacGbl, userCache, groupCache); err != nil {
			return err
		}
	}
	for pID := range r.Projects {
		rbacPrj := &r.Projects[pID]
		if err := fillRBACProjectWithID(ctx, db, rbacPrj, userCache, groupCache); err != nil {
			return err
		}
	}
	return nil
}

func fillRBACProjectWithID(ctx context.Context, db gorp.SqlExecutor, rbacPrj *sdk.RBACProject, userCache map[string]string, groupCache map[string]int64) error {
	rbacPrj.RBACUsersIDs = make([]string, 0, len(rbacPrj.RBACUsersName))
	for _, userName := range rbacPrj.RBACUsersName {
		userID := userCache[userName]
		if userID == "" {
			authentifierUser, err := user.LoadByUsername(ctx, db, userName)
			if err != nil {
				return err
			}
			userID = authentifierUser.ID
			userCache[userName] = userID
		}
		rbacPrj.RBACUsersIDs = append(rbacPrj.RBACUsersIDs, userID)
	}
	rbacPrj.RBACGroupsIDs = make([]int64, 0, len(rbacPrj.RBACGroupsName))
	for _, groupName := range rbacPrj.RBACGroupsName {
		groupID := groupCache[groupName]
		if groupID == 0 {
			groupDB, err := group.LoadByName(ctx, db, groupName)
			if err != nil {
				return err
			}
			groupID = groupDB.ID
			groupCache[groupDB.Name] = groupID
		}
		rbacPrj.RBACGroupsIDs = append(rbacPrj.RBACGroupsIDs, groupID)
	}
	return nil
}

func fillRBACGlobalWithID(ctx context.Context, db gorp.SqlExecutor, rbacGbl *sdk.RBACGlobal, userCache map[string]string, groupCache map[string]int64) error {
	rbacGbl.RBACUsersIDs = make([]string, 0, len(rbacGbl.RBACUsersName))
	for _, rbacUserName := range rbacGbl.RBACUsersName {
		userID := userCache[rbacUserName]
		if userID == "" {
			authentifierUser, err := user.LoadByUsername(ctx, db, rbacUserName)
			if err != nil {
				return err
			}
			userID = authentifierUser.ID
			userCache[rbacUserName] = userID
		}
		rbacGbl.RBACUsersIDs = append(rbacGbl.RBACUsersIDs, userID)
	}

	rbacGbl.RBACGroupsIDs = make([]int64, 0, len(rbacGbl.RBACGroupsName))
	for _, groupName := range rbacGbl.RBACGroupsName {
		groupID := groupCache[groupName]
		if groupID == 0 {
			groupDB, err := group.LoadByName(ctx, db, groupName)
			if err != nil {
				return err
			}
			groupID = groupDB.ID
			groupCache[groupDB.Name] = groupID
		}
		rbacGbl.RBACGroupsIDs = append(rbacGbl.RBACGroupsIDs, groupID)
	}
	return nil
}
