package rbac

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

func IsValidRBAC(ctx context.Context, db gorp.SqlExecutor, rbac *sdk.RBAC) error {
	if rbac.Name == "" {
		return sdk.WrapError(sdk.ErrInvalidData, "missing permission name")
	}
	for _, g := range rbac.Global {
		if err := isValidRBACGlobal(rbac.Name, g); err != nil {
			return err
		}
	}
	for _, p := range rbac.Projects {
		if err := isValidRBACProject(rbac.Name, p); err != nil {
			return err
		}
	}
	for _, r := range rbac.Regions {
		if err := isValidRBACRegion(ctx, db, rbac.Name, r); err != nil {
			return err
		}
	}
	for _, h := range rbac.Hatcheries {
		if err := isValidRBACHatchery(rbac.Name, h); err != nil {
			return err
		}
	}
	return nil
}

func isValidRBACGlobal(rbacName string, rg sdk.RBACGlobal) error {
	if len(rg.RBACGroupsIDs) == 0 && len(rg.RBACUsersIDs) == 0 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: missing groups or users on global permission", rbacName)
	}
	if rg.Role == "" {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: role for global permission cannot be empty", rbacName)
	}

	if !sdk.IsInArray(rg.Role, sdk.GlobalRoles) {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: role %s is not allowed on a global permission", rbacName, rg.Role)
	}
	return nil
}

func isValidRBACProject(rbacName string, rbacProject sdk.RBACProject) error {
	// Check empty group and users
	if len(rbacProject.RBACGroupsIDs) == 0 && len(rbacProject.RBACUsersIDs) == 0 && !rbacProject.AllUsers {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: missing groups or users on project permission", rbacName)
	}

	if (len(rbacProject.RBACGroupsIDs) > 0 || len(rbacProject.RBACUsersIDs) > 0) && rbacProject.AllUsers {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: cannot have a list of groups or users with flag allUsers", rbacName)
	}

	// Check role
	if rbacProject.Role == "" {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: role for project permission cannot be empty", rbacName)
	}
	roleFound := false
	for _, r := range sdk.ProjectRoles {
		if r == rbacProject.Role {
			roleFound = true
			break
		}
	}
	if !roleFound {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: role %s is not allowed on a project permission", rbacName, rbacProject.Role)
	}
	return nil
}

func isValidRBACRegion(ctx context.Context, db gorp.SqlExecutor, rbacName string, rbacRegion sdk.RBACRegion) error {
	if rbacRegion.RegionID == "" {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: missing region", rbacName)
	}

	// Check role
	if rbacRegion.Role == "" {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: role for region permission cannot be empty", rbacName)
	}
	if !sdk.RegionRoles.Contains(rbacRegion.Role) {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: role %s is not allowed on a region permission", rbacName, rbacRegion.Role)
	}

	// Check Organization
	if len(rbacRegion.RBACOrganizationIDs) == 0 {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: must have at least one organization", rbacName)
	}

	if len(rbacRegion.RBACGroupsIDs)+len(rbacRegion.RBACUsersIDs) == 0 && !rbacRegion.AllUsers {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: missing groups or users on region permission", rbacName)
	}
	if len(rbacRegion.RBACGroupsIDs)+len(rbacRegion.RBACUsersIDs) > 0 && rbacRegion.AllUsers {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: you can't have a list of groups/users and the all Users flag checked on a region permission", rbacName)
	}

	// Load organizations
	orgs, err := organization.LoadOrganizationByIDs(ctx, db, rbacRegion.RBACOrganizationIDs)
	if err != nil {
		return err
	}
	orgsIDs := make(map[string]struct{})
	for _, o := range orgs {
		orgsIDs[o.ID] = struct{}{}
	}

	// Check group organization
	if len(rbacRegion.RBACGroupsIDs) > 0 {
		groupOrgs, err := group.LoadGroupOrganizationsByGroupIDs(ctx, db, rbacRegion.RBACGroupsIDs)
		if err != nil {
			return err
		}
		for _, grpOrg := range groupOrgs {
			if _, has := orgsIDs[grpOrg.OrganizationID]; !has {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: some groups are not part of the allowed organizations", rbacName)
			}
		}
	}
	if len(rbacRegion.RBACUsersIDs) > 0 {
		userOrgs, err := user.LoadAllUserOrganizationsByUserIDs(ctx, db, rbacRegion.RBACUsersIDs)
		if err != nil {
			return err
		}
		for _, usrOrg := range userOrgs {
			if _, has := orgsIDs[usrOrg.OrganizationID]; !has {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: some users are not part of the allowed organizations", rbacName)
			}
		}
	}

	return nil
}

func isValidRBACHatchery(rbacName string, rbacHatchery sdk.RBACHatchery) error {
	// Check empty hatchery
	if rbacHatchery.HatcheryID == "" {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: missing hatchery", rbacName)
	}
	if rbacHatchery.RegionID == "" {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: missing region", rbacName)
	}

	// Check role
	if rbacHatchery.Role == "" {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: role for hatchery permission cannot be empty", rbacName)
	}
	if !sdk.HatcheryRoles.Contains(rbacHatchery.Role) {
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "rbac %s: role %s is not allowed on a hatchery permission", rbacName, rbacHatchery.Role)
	}
	return nil
}
