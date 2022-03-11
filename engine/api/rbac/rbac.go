package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func fillWithIDs(ctx context.Context, db gorp.SqlExecutor, r *sdk.Rbac) error {
	// Check existing permission
	uuid, err := LoadRbacUUIDByName(ctx, db, r.Name)
	if err != nil {
		return err
	}
	r.UUID = uuid

	userCache := make(map[string]string)
	groupCache := make(map[string]int64)
	projectCache := make(map[string]int64)
	for gID := range r.Globals {
		rbacGbl := &r.Globals[gID]
		if err := fillRbacGlobalWithID(ctx, db, rbacGbl, userCache, groupCache); err != nil {
			return err
		}
	}
	for pID := range r.Projects {
		rbacPrj := &r.Projects[pID]
		if err := fillRbacProjectWithID(ctx, db, rbacPrj, projectCache, userCache, groupCache); err != nil {
			return err
		}
	}
	return nil
}

func fillRbacProjectWithID(ctx context.Context, db gorp.SqlExecutor, rbacPrj *sdk.RbacProject, projectCache map[string]int64, userCache map[string]string, groupCache map[string]int64) error {
	rbacPrj.RbacProjectsIDs = make([]int64, 0, len(rbacPrj.RbacProjectKeys))
	for _, projKey := range rbacPrj.RbacProjectKeys {
		projectID := projectCache[projKey]
		if projectID == 0 {
			prj, err := project.Load(ctx, db, projKey)
			if err != nil {
				return err
			}
			projectID = prj.ID
			projectCache[projKey] = prj.ID
		}
		rbacPrj.RbacProjectsIDs = append(rbacPrj.RbacProjectsIDs, projectID)
	}
	rbacPrj.RbacUsersIDs = make([]string, 0, len(rbacPrj.RbacUsersName))
	for _, userName := range rbacPrj.RbacUsersName {
		userID := userCache[userName]
		if userID == "" {
			authentifierUser, err := user.LoadByUsername(ctx, db, userName)
			if err != nil {
				return err
			}
			userID = authentifierUser.ID
			userCache[userName] = userID
		}
		rbacPrj.RbacUsersIDs = append(rbacPrj.RbacUsersIDs, userID)
	}
	rbacPrj.RbacGroupsIDs = make([]int64, 0, len(rbacPrj.RbacGroupsName))
	for _, groupName := range rbacPrj.RbacGroupsName {
		groupID := groupCache[groupName]
		if groupID == 0 {
			groupDB, err := group.LoadByName(ctx, db, groupName)
			if err != nil {
				return err
			}
			groupID = groupDB.ID
			groupCache[groupDB.Name] = groupID
		}
		rbacPrj.RbacGroupsIDs = append(rbacPrj.RbacGroupsIDs, groupID)
	}
	return nil
}

func fillRbacGlobalWithID(ctx context.Context, db gorp.SqlExecutor, rbacGbl *sdk.RbacGlobal, userCache map[string]string, groupCache map[string]int64) error {
	rbacGbl.RbacUsersIDs = make([]string, 0, len(rbacGbl.RbacUsersName))
	for _, rbacUserName := range rbacGbl.RbacUsersName {
		userID := userCache[rbacUserName]
		if userID == "" {
			authentifierUser, err := user.LoadByUsername(ctx, db, rbacUserName)
			if err != nil {
				return err
			}
			userID = authentifierUser.ID
			userCache[rbacUserName] = userID
		}
		rbacGbl.RbacUsersIDs = append(rbacGbl.RbacUsersIDs, userID)
	}

	rbacGbl.RbacGroupsIDs = make([]int64, 0, len(rbacGbl.RbacGroupsName))
	for _, groupName := range rbacGbl.RbacGroupsName {
		groupID := groupCache[groupName]
		if groupID == 0 {
			groupDB, err := group.LoadByName(ctx, db, groupName)
			if err != nil {
				return err
			}
			groupID = groupDB.ID
			groupCache[groupDB.Name] = groupID
		}
		rbacGbl.RbacGroupsIDs = append(rbacGbl.RbacGroupsIDs, groupID)
	}
	return nil
}
