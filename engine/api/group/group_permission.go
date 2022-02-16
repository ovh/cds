package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func CheckGroupPermission(ctx context.Context, db gorp.SqlExecutor, projectGroups sdk.GroupPermissions, gp *sdk.GroupPermission, consumer *sdk.AuthConsumer) error {
	if gp.Group.ID == 0 {
		g, err := LoadByName(ctx, db, gp.Group.Name)
		if err != nil {
			return err
		}
		gp.Group = *g
	}
	if err := LoadOptions.WithMembers(ctx, db, &gp.Group); err != nil {
		return err
	}

	if IsDefaultGroupID(gp.Group.ID) && gp.Permission > sdk.PermissionRead {
		return sdk.NewErrorFrom(sdk.ErrDefaultGroupPermission, "only read permission is allowed to default group")
	}

	if err := CheckGroupPermissionOrganizationMatch(ctx, db, projectGroups, &gp.Group, gp.Permission); err != nil {
		return err
	}

	if !IsConsumerGroupAdmin(&gp.Group, consumer) && gp.Permission > sdk.PermissionRead {
		return sdk.WithStack(sdk.ErrInvalidGroupAdmin)
	}

	return nil
}

func CheckProjectOrganizationMatch(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project, grp *sdk.Group, role int) error {
	if err := LoadGroupsIntoProject(ctx, db, proj); err != nil {
		return err
	}
	return CheckGroupPermissionOrganizationMatch(ctx, db, proj.ProjectGroups, grp, role)
}

func CheckGroupPermissionOrganizationMatch(ctx context.Context, db gorp.SqlExecutor, projectGroups sdk.GroupPermissions, grp *sdk.Group, role int) error {
	if role == sdk.PermissionRead {
		return nil
	}

	projectOrganization, err := projectGroups.ComputeOrganization()
	if err != nil {
		return sdk.NewError(sdk.ErrForbidden, err)
	}
	if projectOrganization == "" {
		return nil
	}

	if err := LoadOptions.WithOrganization(ctx, db, grp); err != nil {
		return err
	}
	if grp.Organization != projectOrganization {
		if grp.Organization == "" {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "given group without organization don't match project organization %q", projectOrganization)
		}
		return sdk.NewErrorFrom(sdk.ErrForbidden, "given group with organization %q don't match project organization %q", grp.Organization, projectOrganization)
	}

	return nil
}
