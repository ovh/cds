package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/service"
)

func (api *API) getRbacByIdentifier(ctx context.Context, rbacIdentifier string, opts ...rbac.LoadOptionFunc) (*sdk.RBAC, error) {
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

func (api *API) getRbacHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalPermissionManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			rbacIdentifier := vars["rbacIdentifier"]
			perm, err := api.getRbacByIdentifier(ctx, rbacIdentifier,
				rbac.LoadOptions.LoadRBACGlobal,
				rbac.LoadOptions.LoadRBACProject,
				rbac.LoadOptions.LoadRBACHatchery,
				rbac.LoadOptions.LoadRBACRegion)
			if err != nil {
				return err
			}

			if err := api.FillRBACWithNames(ctx, perm); err != nil {
				return err
			}

			return service.WriteMarshal(w, req, perm, http.StatusOK)
		}
}

func (api *API) deleteRbacHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalPermissionManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			rbacIdentifier := vars["rbacIdentifier"]

			perm, err := api.getRbacByIdentifier(ctx, rbacIdentifier)
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
			return service.WriteMarshal(w, req, nil, http.StatusOK)
		}
}

func (api *API) postImportRbacHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.globalPermissionManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			force := service.FormBool(req, "force")

			var rbacRule sdk.RBAC
			if err := service.UnmarshalRequest(ctx, req, &rbacRule); err != nil {
				return err
			}

			existingRule, err := rbac.LoadRBACByName(ctx, api.mustDB(), rbacRule.Name)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}

			if err := api.FillRBACWithIDs(ctx, &rbacRule); err != nil {
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
			return service.WriteMarshal(w, req, nil, http.StatusCreated)
		}
}

func (a *API) FillRBACWithNames(ctx context.Context, r *sdk.RBAC) error {
	userCache := make(map[string]string)
	groupCache := make(map[int64]string)
	organizationCache := make(map[string]string)

	for gID := range r.Globals {
		rbacGbl := &r.Globals[gID]
		if err := a.fillRBACGlobalWithNames(ctx, rbacGbl, userCache, groupCache); err != nil {
			return err
		}
	}
	for pID := range r.Projects {
		rbacPrj := &r.Projects[pID]
		if err := a.fillRBACProjectWithNames(ctx, rbacPrj, userCache, groupCache); err != nil {
			return err
		}
	}
	for rID := range r.Regions {
		rbacRg := &r.Regions[rID]
		if err := a.fillRBACRegionWithNames(ctx, rbacRg, userCache, groupCache, organizationCache); err != nil {
			return err
		}
	}
	for hID := range r.Hatcheries {
		rbacOrg := &r.Hatcheries[hID]
		if err := a.fillRBACHatcheryWithNames(ctx, rbacOrg); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) FillRBACWithIDs(ctx context.Context, r *sdk.RBAC) error {
	// Check existing permission
	rbacDB, err := rbac.LoadRBACByName(ctx, a.mustDB(), r.Name)
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
	organizationCache := make(map[string]string)
	for gID := range r.Globals {
		rbacGbl := &r.Globals[gID]
		if err := a.fillRBACGlobalWithID(ctx, rbacGbl, userCache, groupCache); err != nil {
			return err
		}
	}
	for pID := range r.Projects {
		rbacPrj := &r.Projects[pID]
		if err := a.fillRBACProjectWithID(ctx, rbacPrj, userCache, groupCache); err != nil {
			return err
		}
	}
	for rID := range r.Regions {
		rbacRg := &r.Regions[rID]
		if err := a.fillRBACRegionWithID(ctx, rbacRg, userCache, groupCache, organizationCache); err != nil {
			return err
		}
	}
	for hID := range r.Hatcheries {
		rbacOrg := &r.Hatcheries[hID]
		if err := a.fillRBACHatcheryWithID(ctx, rbacOrg); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) fillRBACHatcheryWithNames(ctx context.Context, rbacHatchery *sdk.RBACHatchery) error {
	hatch, err := hatchery.LoadHatcheryByID(ctx, a.mustDB(), rbacHatchery.HatcheryID)
	if err != nil {
		return err
	}
	rbacHatchery.HatcheryName = hatch.Name

	reg, err := region.LoadRegionByID(ctx, a.mustDB(), rbacHatchery.RegionID)
	if err != nil {
		return err
	}
	rbacHatchery.RegionName = reg.Name
	return nil
}

func (a *API) fillRBACHatcheryWithID(ctx context.Context, rbacHatchery *sdk.RBACHatchery) error {
	hatch, err := hatchery.LoadHatcheryByName(ctx, a.mustDB(), rbacHatchery.HatcheryName)
	if err != nil {
		return err
	}
	rbacHatchery.HatcheryID = hatch.ID

	reg, err := region.LoadRegionByName(ctx, a.mustDB(), rbacHatchery.RegionName)
	if err != nil {
		return err
	}
	rbacHatchery.RegionID = reg.ID
	return nil
}

func (a *API) fillRBACRegionWithNames(ctx context.Context, rbacRegion *sdk.RBACRegion, userCache map[string]string, groupCache map[int64]string, organizationCache map[string]string) error {
	rg, err := region.LoadRegionByID(ctx, a.mustDB(), rbacRegion.RegionID)
	if err != nil {
		return err
	}
	rbacRegion.RegionName = rg.Name

	rbacRegion.RBACUsersName = make([]string, 0, len(rbacRegion.RBACUsersIDs))
	for _, userID := range rbacRegion.RBACUsersIDs {
		userName := userCache[userID]
		if userName == "" {
			authentifierUser, err := user.LoadByID(ctx, a.mustDB(), userID)
			if err != nil {
				return err
			}
			userName = authentifierUser.Username
			userCache[userID] = userName
		}
		rbacRegion.RBACUsersName = append(rbacRegion.RBACUsersName, userName)
	}

	rbacRegion.RBACGroupsName = make([]string, 0, len(rbacRegion.RBACGroupsIDs))
	for _, groupID := range rbacRegion.RBACGroupsIDs {
		groupName := groupCache[groupID]
		if groupID == 0 {
			groupDB, err := group.LoadByID(ctx, a.mustDB(), groupID)
			if err != nil {
				return err
			}
			groupName = groupDB.Name
			groupCache[groupDB.ID] = groupName
		}
		rbacRegion.RBACGroupsName = append(rbacRegion.RBACGroupsName, groupName)
	}

	rbacRegion.RBACOrganizations = make([]string, 0, len(rbacRegion.RBACOrganizationIDs))
	for _, orgID := range rbacRegion.RBACOrganizationIDs {
		orgName := organizationCache[orgID]
		if orgName == "" {
			orgDB, err := organization.LoadOrganizationByID(ctx, a.mustDB(), orgID)
			if err != nil {
				return err
			}
			orgName = orgDB.Name
			organizationCache[orgDB.ID] = orgName
		}
		rbacRegion.RBACOrganizations = append(rbacRegion.RBACOrganizations, orgName)
	}
	return nil
}

func (a *API) fillRBACRegionWithID(ctx context.Context, rbacRegion *sdk.RBACRegion, userCache map[string]string, groupCache map[string]int64, organizationCache map[string]string) error {
	rg, err := region.LoadRegionByName(ctx, a.mustDB(), rbacRegion.RegionName)
	if err != nil {
		return err
	}
	rbacRegion.RegionID = rg.ID

	rbacRegion.RBACUsersIDs = make([]string, 0, len(rbacRegion.RBACUsersName))
	for _, userName := range rbacRegion.RBACUsersName {
		userID := userCache[userName]
		if userID == "" {
			authentifierUser, err := user.LoadByUsername(ctx, a.mustDB(), userName)
			if err != nil {
				return err
			}
			userID = authentifierUser.ID
			userCache[userName] = userID
		}
		rbacRegion.RBACUsersIDs = append(rbacRegion.RBACUsersIDs, userID)
	}

	rbacRegion.RBACGroupsIDs = make([]int64, 0, len(rbacRegion.RBACGroupsName))
	for _, groupName := range rbacRegion.RBACGroupsName {
		groupID := groupCache[groupName]
		if groupID == 0 {
			groupDB, err := group.LoadByName(ctx, a.mustDB(), groupName)
			if err != nil {
				return err
			}
			groupID = groupDB.ID
			groupCache[groupDB.Name] = groupID
		}
		rbacRegion.RBACGroupsIDs = append(rbacRegion.RBACGroupsIDs, groupID)
	}

	rbacRegion.RBACOrganizationIDs = make([]string, 0, len(rbacRegion.RBACOrganizations))
	for _, orgaName := range rbacRegion.RBACOrganizations {
		orgID := organizationCache[orgaName]
		if orgID == "" {
			orgDB, err := organization.LoadOrganizationByName(ctx, a.mustDB(), orgaName)
			if err != nil {
				return err
			}
			orgID = orgDB.ID
			organizationCache[orgDB.Name] = orgID
		}
		rbacRegion.RBACOrganizationIDs = append(rbacRegion.RBACOrganizationIDs, orgID)
	}
	return nil
}

func (a *API) fillRBACProjectWithNames(ctx context.Context, rbacPrj *sdk.RBACProject, userCache map[string]string, groupCache map[int64]string) error {
	rbacPrj.RBACUsersName = make([]string, 0, len(rbacPrj.RBACUsersIDs))
	for _, userID := range rbacPrj.RBACUsersIDs {
		userName := userCache[userID]
		if userName == "" {
			authentifierUser, err := user.LoadByID(ctx, a.mustDB(), userID)
			if err != nil {
				return err
			}
			userName = authentifierUser.Username
			userCache[userID] = userName
		}
		rbacPrj.RBACUsersName = append(rbacPrj.RBACUsersName, userName)
	}
	rbacPrj.RBACGroupsName = make([]string, 0, len(rbacPrj.RBACGroupsIDs))
	for _, groupID := range rbacPrj.RBACGroupsIDs {
		groupName := groupCache[groupID]
		if groupName == "" {
			groupDB, err := group.LoadByID(ctx, a.mustDB(), groupID)
			if err != nil {
				return err
			}
			groupName = groupDB.Name
			groupCache[groupDB.ID] = groupName
		}
		rbacPrj.RBACGroupsName = append(rbacPrj.RBACGroupsName, groupName)
	}
	return nil
}

func (a *API) fillRBACProjectWithID(ctx context.Context, rbacPrj *sdk.RBACProject, userCache map[string]string, groupCache map[string]int64) error {
	rbacPrj.RBACUsersIDs = make([]string, 0, len(rbacPrj.RBACUsersName))
	for _, userName := range rbacPrj.RBACUsersName {
		userID := userCache[userName]
		if userID == "" {
			authentifierUser, err := user.LoadByUsername(ctx, a.mustDB(), userName)
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
			groupDB, err := group.LoadByName(ctx, a.mustDB(), groupName)
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

func (a *API) fillRBACGlobalWithNames(ctx context.Context, rbacGbl *sdk.RBACGlobal, userCache map[string]string, groupCache map[int64]string) error {

	rbacGbl.RBACUsersName = make([]string, 0, len(rbacGbl.RBACUsersIDs))
	for _, rbacUserID := range rbacGbl.RBACUsersIDs {
		userName := userCache[rbacUserID]
		if userName == "" {
			authentifierUser, err := user.LoadByID(ctx, a.mustDB(), rbacUserID)
			if err != nil {
				return err
			}
			userName = authentifierUser.Username
			userCache[rbacUserID] = userName
		}
		rbacGbl.RBACUsersName = append(rbacGbl.RBACUsersName, userName)
	}

	rbacGbl.RBACGroupsName = make([]string, 0, len(rbacGbl.RBACGroupsIDs))
	for _, groupID := range rbacGbl.RBACGroupsIDs {
		groupName := groupCache[groupID]
		if groupName == "" {
			groupDB, err := group.LoadByID(ctx, a.mustDB(), groupID)
			if err != nil {
				return err
			}
			groupName = groupDB.Name
			groupCache[groupDB.ID] = groupName
		}
		rbacGbl.RBACGroupsName = append(rbacGbl.RBACGroupsName, groupName)
	}
	return nil
}

func (a *API) fillRBACGlobalWithID(ctx context.Context, rbacGbl *sdk.RBACGlobal, userCache map[string]string, groupCache map[string]int64) error {
	rbacGbl.RBACUsersIDs = make([]string, 0, len(rbacGbl.RBACUsersName))
	for _, rbacUserName := range rbacGbl.RBACUsersName {
		userID := userCache[rbacUserName]
		if userID == "" {
			authentifierUser, err := user.LoadByUsername(ctx, a.mustDB(), rbacUserName)
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
			groupDB, err := group.LoadByName(ctx, a.mustDB(), groupName)
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
