package api

import (
	"context"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
)

type RBACLoader struct {
	db                gorp.SqlExecutor
	userCache         map[string]string
	groupCache        map[int64]string
	groupIDCache      map[string]int64
	organizationCache map[string]string
}

func NewRBACLoader(db gorp.SqlExecutor) *RBACLoader {
	return &RBACLoader{
		db:                db,
		userCache:         make(map[string]string),
		groupCache:        make(map[int64]string),
		groupIDCache:      make(map[string]int64),
		organizationCache: make(map[string]string),
	}
}

func (api *API) getRBACByIdentifier(ctx context.Context, rbacIdentifier string, opts ...rbac.LoadOptionFunc) (*sdk.RBAC, error) {
	var repo *sdk.RBAC
	var err error
	if sdk.IsValidUUID(rbacIdentifier) {
		repo, err = rbac.LoadRBACByID(ctx, api.mustDB(), rbacIdentifier, opts...)
	} else {
		repo, err = rbac.LoadRBACByName(ctx, api.mustDB(), rbacIdentifier, opts...)
	}
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (api *API) getPermissionsHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalPermissionManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			perms, err := rbac.LoadAll(ctx, api.mustDB())
			if err != nil {
				return err
			}
			return service.WriteJSON(w, perms, http.StatusOK)
		}
}

func (api *API) getRBACHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalPermissionManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			rbacIdentifier := vars["rbacIdentifier"]
			perm, err := api.getRBACByIdentifier(ctx, rbacIdentifier,
				rbac.LoadOptions.LoadRBACGlobal,
				rbac.LoadOptions.LoadRBACProject,
				rbac.LoadOptions.LoadRBACWorkflow,
				rbac.LoadOptions.LoadRBACHatchery,
				rbac.LoadOptions.LoadRBACRegion)
			if err != nil {
				return err
			}

			rbacLoader := NewRBACLoader(api.mustDB())
			if err := rbacLoader.FillRBACWithNames(ctx, perm); err != nil {
				return err
			}

			return service.WriteMarshal(w, req, perm, http.StatusOK)
		}
}

func (api *API) deleteRBACHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalPermissionManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			rbacIdentifier := vars["rbacIdentifier"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			perm, err := api.getRBACByIdentifier(ctx, rbacIdentifier)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := rbac.Delete(ctx, tx, *perm); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			event_v2.PublishPermissionEvent(ctx, api.Cache, sdk.EventPermissionDeleted, *perm, *u.AuthConsumerUser.AuthentifiedUser)
			return service.WriteMarshal(w, req, nil, http.StatusOK)
		}
}

func (api *API) postImportRBACHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalPermissionManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			force := service.FormBool(req, "force")

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var rbacRule sdk.RBAC
			if err := service.UnmarshalRequest(ctx, req, &rbacRule); err != nil {
				return err
			}

			existingRule, err := rbac.LoadRBACByName(ctx, api.mustDB(), rbacRule.Name)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}

			rbacLoader := NewRBACLoader(api.mustDB())
			if err := rbacLoader.FillRBACWithIDs(ctx, &rbacRule); err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if existingRule != nil && !force {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "unable to override existing permission")
			}
			if existingRule != nil {
				if err := rbac.Delete(ctx, tx, *existingRule); err != nil {
					return err
				}
			}

			if err := rbac.Insert(ctx, tx, &rbacRule); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			if existingRule == nil {
				event_v2.PublishPermissionEvent(ctx, api.Cache, sdk.EventPermissionCreated, rbacRule, *u.AuthConsumerUser.AuthentifiedUser)
			} else {
				event_v2.PublishPermissionEvent(ctx, api.Cache, sdk.EventPermissionUpdated, rbacRule, *u.AuthConsumerUser.AuthentifiedUser)
			}
			return service.WriteMarshal(w, req, nil, http.StatusCreated)
		}
}

func (rl *RBACLoader) FillRBACWithNames(ctx context.Context, r *sdk.RBAC) error {
	for gID := range r.Global {
		rbacGbl := &r.Global[gID]
		if err := rl.fillRBACGlobalWithNames(ctx, rbacGbl); err != nil {
			return err
		}
	}
	for pID := range r.Projects {
		rbacPrj := &r.Projects[pID]
		if err := rl.fillRBACProjectWithNames(ctx, rbacPrj); err != nil {
			return err
		}
	}
	for rID := range r.Regions {
		rbacRg := &r.Regions[rID]
		if err := rl.fillRBACRegionWithNames(ctx, rbacRg); err != nil {
			return err
		}
	}
	for hID := range r.Hatcheries {
		rbacOrg := &r.Hatcheries[hID]
		if err := rl.fillRBACHatcheryWithNames(ctx, rbacOrg); err != nil {
			return err
		}
	}
	for wID := range r.Workflows {
		rbacWkf := &r.Workflows[wID]
		if err := rl.fillRBACWorkflowWithNames(ctx, rbacWkf); err != nil {
			return err
		}
	}
	return nil
}

func (rl *RBACLoader) FillRBACWithIDs(ctx context.Context, r *sdk.RBAC) error {
	// Check existing permission
	rbacDB, err := rbac.LoadRBACByName(ctx, rl.db, r.Name)
	if err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
	}
	if rbacDB != nil {
		r.ID = rbacDB.ID
	}

	for gID := range r.Global {
		rbacGbl := &r.Global[gID]
		if err := rl.fillRBACGlobalWithID(ctx, rbacGbl); err != nil {
			return err
		}
	}
	for pID := range r.Projects {
		rbacPrj := &r.Projects[pID]
		if err := rl.fillRBACProjectWithID(ctx, rbacPrj); err != nil {
			return err
		}
	}
	for rID := range r.Regions {
		rbacRg := &r.Regions[rID]
		if err := rl.fillRBACRegionWithID(ctx, rbacRg); err != nil {
			return err
		}
	}
	for hID := range r.Hatcheries {
		rbacOrg := &r.Hatcheries[hID]
		if err := rl.fillRBACHatcheryWithID(ctx, rbacOrg); err != nil {
			return err
		}
	}
	for wID := range r.Workflows {
		rbacWf := &r.Workflows[wID]
		if err := rl.fillRBACWorkflowWIthID(ctx, rbacWf); err != nil {
			return err
		}
	}
	return nil
}

func (rl *RBACLoader) fillRBACHatcheryWithNames(ctx context.Context, rbacHatchery *sdk.RBACHatchery) error {
	hatch, err := hatchery.LoadHatcheryByID(ctx, rl.db, rbacHatchery.HatcheryID)
	if err != nil {
		return err
	}
	rbacHatchery.HatcheryName = hatch.Name

	reg, err := region.LoadRegionByID(ctx, rl.db, rbacHatchery.RegionID)
	if err != nil {
		return err
	}
	rbacHatchery.RegionName = reg.Name
	return nil
}

func (rl *RBACLoader) fillRBACHatcheryWithID(ctx context.Context, rbacHatchery *sdk.RBACHatchery) error {
	hatch, err := hatchery.LoadHatcheryByName(ctx, rl.db, rbacHatchery.HatcheryName)
	if err != nil {
		return err
	}
	rbacHatchery.HatcheryID = hatch.ID

	reg, err := region.LoadRegionByName(ctx, rl.db, rbacHatchery.RegionName)
	if err != nil {
		return err
	}
	rbacHatchery.RegionID = reg.ID
	return nil
}

func (rl *RBACLoader) fillRBACRegionWithNames(ctx context.Context, rbacRegion *sdk.RBACRegion) error {
	rg, err := region.LoadRegionByID(ctx, rl.db, rbacRegion.RegionID)
	if err != nil {
		return err
	}
	rbacRegion.RegionName = rg.Name

	rbacRegion.RBACUsersName = make([]string, 0, len(rbacRegion.RBACUsersIDs))
	for _, userID := range rbacRegion.RBACUsersIDs {
		userName := rl.userCache[userID]
		if userName == "" {
			authentifierUser, err := user.LoadByID(ctx, rl.db, userID)
			if err != nil {
				return err
			}
			userName = authentifierUser.Username
			rl.userCache[userID] = userName
		}
		rbacRegion.RBACUsersName = append(rbacRegion.RBACUsersName, userName)
	}

	rbacRegion.RBACGroupsName = make([]string, 0, len(rbacRegion.RBACGroupsIDs))
	for _, groupID := range rbacRegion.RBACGroupsIDs {
		groupName := rl.groupCache[groupID]
		if groupName == "" {
			groupDB, err := group.LoadByID(ctx, rl.db, groupID)
			if err != nil {
				return err
			}
			groupName = groupDB.Name
			rl.groupCache[groupDB.ID] = groupName
		}
		rbacRegion.RBACGroupsName = append(rbacRegion.RBACGroupsName, groupName)
	}

	rbacRegion.RBACOrganizations = make([]string, 0, len(rbacRegion.RBACOrganizationIDs))
	for _, orgID := range rbacRegion.RBACOrganizationIDs {
		orgName := rl.organizationCache[orgID]
		if orgName == "" {
			orgDB, err := organization.LoadOrganizationByID(ctx, rl.db, orgID)
			if err != nil {
				return err
			}
			orgName = orgDB.Name
			rl.organizationCache[orgDB.ID] = orgName
		}
		rbacRegion.RBACOrganizations = append(rbacRegion.RBACOrganizations, orgName)
	}
	return nil
}

func (rl *RBACLoader) fillRBACRegionWithID(ctx context.Context, rbacRegion *sdk.RBACRegion) error {
	rg, err := region.LoadRegionByName(ctx, rl.db, rbacRegion.RegionName)
	if err != nil {
		return err
	}
	rbacRegion.RegionID = rg.ID

	rbacRegion.RBACUsersIDs = make([]string, 0, len(rbacRegion.RBACUsersName))
	for _, userName := range rbacRegion.RBACUsersName {
		userID := rl.userCache[userName]
		if userID == "" {
			authentifierUser, err := user.LoadByUsername(ctx, rl.db, userName)
			if err != nil {
				return err
			}
			userID = authentifierUser.ID
			rl.userCache[userName] = userID
		}
		rbacRegion.RBACUsersIDs = append(rbacRegion.RBACUsersIDs, userID)
	}

	rbacRegion.RBACGroupsIDs = make([]int64, 0, len(rbacRegion.RBACGroupsName))
	for _, groupName := range rbacRegion.RBACGroupsName {
		groupID := rl.groupIDCache[groupName]
		if groupID == 0 {
			groupDB, err := group.LoadByName(ctx, rl.db, groupName)
			if err != nil {
				return err
			}
			groupID = groupDB.ID
			rl.groupIDCache[groupDB.Name] = groupID
		}
		rbacRegion.RBACGroupsIDs = append(rbacRegion.RBACGroupsIDs, groupID)
	}

	rbacRegion.RBACOrganizationIDs = make([]string, 0, len(rbacRegion.RBACOrganizations))
	for _, orgaName := range rbacRegion.RBACOrganizations {
		orgID := rl.organizationCache[orgaName]
		if orgID == "" {
			orgDB, err := organization.LoadOrganizationByName(ctx, rl.db, orgaName)
			if err != nil {
				return err
			}
			orgID = orgDB.ID
			rl.organizationCache[orgDB.Name] = orgID
		}
		rbacRegion.RBACOrganizationIDs = append(rbacRegion.RBACOrganizationIDs, orgID)
	}
	return nil
}

func (rl *RBACLoader) fillRBACProjectWithNames(ctx context.Context, rbacPrj *sdk.RBACProject) error {
	rbacPrj.RBACUsersName = make([]string, 0, len(rbacPrj.RBACUsersIDs))
	for _, userID := range rbacPrj.RBACUsersIDs {
		userName := rl.userCache[userID]
		if userName == "" {
			authentifierUser, err := user.LoadByID(ctx, rl.db, userID)
			if err != nil {
				return err
			}
			userName = authentifierUser.Username
			rl.userCache[userID] = userName
		}
		rbacPrj.RBACUsersName = append(rbacPrj.RBACUsersName, userName)
	}
	rbacPrj.RBACGroupsName = make([]string, 0, len(rbacPrj.RBACGroupsIDs))
	for _, groupID := range rbacPrj.RBACGroupsIDs {
		groupName := rl.groupCache[groupID]
		if groupName == "" {
			groupDB, err := group.LoadByID(ctx, rl.db, groupID)
			if err != nil {
				return err
			}
			groupName = groupDB.Name
			rl.groupCache[groupDB.ID] = groupName
		}
		rbacPrj.RBACGroupsName = append(rbacPrj.RBACGroupsName, groupName)
	}
	return nil
}

func (rl *RBACLoader) fillRBACProjectWithID(ctx context.Context, rbacPrj *sdk.RBACProject) error {
	rbacPrj.RBACUsersIDs = make([]string, 0, len(rbacPrj.RBACUsersName))
	for _, userName := range rbacPrj.RBACUsersName {
		userID := rl.userCache[userName]
		if userID == "" {
			authentifierUser, err := user.LoadByUsername(ctx, rl.db, userName)
			if err != nil {
				return err
			}
			userID = authentifierUser.ID
			rl.userCache[userName] = userID
		}
		rbacPrj.RBACUsersIDs = append(rbacPrj.RBACUsersIDs, userID)
	}
	rbacPrj.RBACGroupsIDs = make([]int64, 0, len(rbacPrj.RBACGroupsName))
	for _, groupName := range rbacPrj.RBACGroupsName {
		groupID := rl.groupIDCache[groupName]
		if groupID == 0 {
			groupDB, err := group.LoadByName(ctx, rl.db, groupName)
			if err != nil {
				return err
			}
			groupID = groupDB.ID
			rl.groupIDCache[groupDB.Name] = groupID
		}
		rbacPrj.RBACGroupsIDs = append(rbacPrj.RBACGroupsIDs, groupID)
	}
	return nil
}

func (rl *RBACLoader) fillRBACWorkflowWithNames(ctx context.Context, rbacWkf *sdk.RBACWorkflow) error {
	rbacWkf.RBACUsersName = make([]string, 0, len(rbacWkf.RBACUsersIDs))
	for _, userID := range rbacWkf.RBACUsersIDs {
		userName := rl.userCache[userID]
		if userName == "" {
			authentifierUser, err := user.LoadByID(ctx, rl.db, userID)
			if err != nil {
				return err
			}
			userName = authentifierUser.Username
			rl.userCache[userID] = userName
		}
		rbacWkf.RBACUsersName = append(rbacWkf.RBACUsersName, userName)
	}
	rbacWkf.RBACGroupsName = make([]string, 0, len(rbacWkf.RBACGroupsIDs))
	for _, groupID := range rbacWkf.RBACGroupsIDs {
		groupName := rl.groupCache[groupID]
		if groupName == "" {
			groupDB, err := group.LoadByID(ctx, rl.db, groupID)
			if err != nil {
				return err
			}
			groupName = groupDB.Name
			rl.groupCache[groupDB.ID] = groupName
		}
		rbacWkf.RBACGroupsName = append(rbacWkf.RBACGroupsName, groupName)
	}
	return nil
}

func (rl *RBACLoader) fillRBACWorkflowWIthID(ctx context.Context, rbacWkf *sdk.RBACWorkflow) error {
	rbacWkf.RBACUsersIDs = make([]string, 0, len(rbacWkf.RBACUsersName))
	for _, userName := range rbacWkf.RBACUsersName {
		userID := rl.userCache[userName]
		if userID == "" {
			authentifierUser, err := user.LoadByUsername(ctx, rl.db, userName)
			if err != nil {
				return err
			}
			userID = authentifierUser.ID
			rl.userCache[userName] = userID
		}
		rbacWkf.RBACUsersIDs = append(rbacWkf.RBACUsersIDs, userID)
	}
	rbacWkf.RBACGroupsIDs = make([]int64, 0, len(rbacWkf.RBACGroupsName))
	for _, groupName := range rbacWkf.RBACGroupsName {
		groupID := rl.groupIDCache[groupName]
		if groupID == 0 {
			groupDB, err := group.LoadByName(ctx, rl.db, groupName)
			if err != nil {
				return err
			}
			groupID = groupDB.ID
			rl.groupIDCache[groupDB.Name] = groupID
		}
		rbacWkf.RBACGroupsIDs = append(rbacWkf.RBACGroupsIDs, groupID)
	}
	return nil
}

func (rl *RBACLoader) fillRBACGlobalWithNames(ctx context.Context, rbacGbl *sdk.RBACGlobal) error {

	rbacGbl.RBACUsersName = make([]string, 0, len(rbacGbl.RBACUsersIDs))
	for _, rbacUserID := range rbacGbl.RBACUsersIDs {
		userName := rl.userCache[rbacUserID]
		if userName == "" {
			authentifierUser, err := user.LoadByID(ctx, rl.db, rbacUserID)
			if err != nil {
				return err
			}
			userName = authentifierUser.Username
			rl.userCache[rbacUserID] = userName
		}
		rbacGbl.RBACUsersName = append(rbacGbl.RBACUsersName, userName)
	}

	rbacGbl.RBACGroupsName = make([]string, 0, len(rbacGbl.RBACGroupsIDs))
	for _, groupID := range rbacGbl.RBACGroupsIDs {
		groupName := rl.groupCache[groupID]
		if groupName == "" {
			groupDB, err := group.LoadByID(ctx, rl.db, groupID)
			if err != nil {
				return err
			}
			groupName = groupDB.Name
			rl.groupCache[groupDB.ID] = groupName
		}
		rbacGbl.RBACGroupsName = append(rbacGbl.RBACGroupsName, groupName)
	}
	return nil
}

func (rl *RBACLoader) fillRBACGlobalWithID(ctx context.Context, rbacGbl *sdk.RBACGlobal) error {
	rbacGbl.RBACUsersIDs = make([]string, 0, len(rbacGbl.RBACUsersName))
	for _, rbacUserName := range rbacGbl.RBACUsersName {
		userID := rl.userCache[rbacUserName]
		if userID == "" {
			authentifierUser, err := user.LoadByUsername(ctx, rl.db, rbacUserName)
			if err != nil {
				return err
			}
			userID = authentifierUser.ID
			rl.userCache[rbacUserName] = userID
		}
		rbacGbl.RBACUsersIDs = append(rbacGbl.RBACUsersIDs, userID)
	}

	rbacGbl.RBACGroupsIDs = make([]int64, 0, len(rbacGbl.RBACGroupsName))
	for _, groupName := range rbacGbl.RBACGroupsName {
		groupID := rl.groupIDCache[groupName]
		if groupID == 0 {
			groupDB, err := group.LoadByName(ctx, rl.db, groupName)
			if err != nil {
				return err
			}
			groupID = groupDB.ID
			rl.groupIDCache[groupDB.Name] = groupID
		}
		rbacGbl.RBACGroupsIDs = append(rbacGbl.RBACGroupsIDs, groupID)
	}
	return nil
}
