package group

import (
	"context"
	"github.com/ovh/cds/engine/api/organization"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// Create insert a new group in database and set user for given id as group admin.
func Create(ctx context.Context, db gorpmapper.SqlExecutorWithTx, grp *sdk.Group, user *sdk.AuthentifiedUser) error {
	if err := Insert(ctx, db, grp); err != nil {
		return err
	}

	if err := InsertLinkGroupUser(ctx, db, &LinkGroupUser{
		GroupID:            grp.ID,
		AuthentifiedUserID: user.ID,
		Admin:              true,
	}); err != nil {
		return err
	}

	org, err := organization.LoadOrganizationByName(ctx, db, user.Organization)
	if err != nil {
		return err
	}

	if err := InsertGroupOrganization(ctx, db, &GroupOrganization{
		GroupID:        grp.ID,
		OrganizationID: org.ID,
	}); err != nil {
		return err
	}

	return nil
}

// Create insert a new group in database and set user for given id as group admin.
func Upsert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, oldGroup, newGroup *sdk.Group) error {
	if oldGroup == nil {
		if err := Insert(ctx, db, newGroup); err != nil {
			return err
		}
	} else {
		newGroup.ID = oldGroup.ID
		if err := Update(ctx, db, newGroup); err != nil {
			return err
		}
	}

	if err := newGroup.Members.CheckAdminExists(); err != nil {
		return err
	}

	if oldGroup != nil {
		if err := DeleteAllLinksGroupUserForGroupID(db, oldGroup.ID); err != nil {
			return err
		}
	}

	for i := range newGroup.Members {
		if err := InsertLinkGroupUser(ctx, db, &LinkGroupUser{
			GroupID:            newGroup.ID,
			AuthentifiedUserID: newGroup.Members[i].ID,
			Admin:              newGroup.Members[i].Admin,
		}); err != nil {
			return err
		}
	}

	return nil
}

// Delete deletes group and dependencies.
func Delete(_ context.Context, db gorp.SqlExecutor, g *sdk.Group) error {
	// Remove the group from database, this will also delete cascade group_user links
	return deleteDB(db, g)
}

// EnsureOrganization computes group organization from members list and save it if needed.
func EnsureOrganization(ctx context.Context, db gorpmapper.SqlExecutorWithTx, g *sdk.Group) error {
	if err := LoadOptions.WithMembers(ctx, db, g); err != nil {
		return err
	}
	exitingGroupOrganization, err := LoadGroupOrganizationByGroupID(ctx, db, g.ID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}

	newOrganizationName, err := g.Members.ComputeOrganization()
	if err != nil {
		return sdk.WrapError(err, "unable to validate group %s", g.Name)
	}
	if exitingGroupOrganization != nil {
		currentOrganization, err := organization.LoadOrganizationByID(ctx, db, exitingGroupOrganization.OrganizationID)
		if err != nil {
			return err
		}
		g.Organization = currentOrganization.Name
	}
	if g.Organization == newOrganizationName {
		return nil
	}

	newGroupOrganization, err := organization.LoadOrganizationByName(ctx, db, newOrganizationName)
	if err != nil {
		return err
	}

	g.Organization = newOrganizationName
	if exitingGroupOrganization == nil {
		if err := InsertGroupOrganization(ctx, db, &GroupOrganization{
			GroupID:        g.ID,
			OrganizationID: newGroupOrganization.ID,
		}); err != nil {
			return err
		}
	} else {
		exitingGroupOrganization.OrganizationID = newGroupOrganization.ID
		if err := UpdateGroupOrganization(ctx, db, exitingGroupOrganization); err != nil {
			return err
		}
	}

	// If organization was changed, check that the group org is not in conflict on each projects
	// Load all projects with permission RX or RWX for current group
	links, err := LoadLinksGroupProjectForGroupID(ctx, db, g.ID)
	if err != nil {
		return err
	}
	var projectIDs []int64
	for i := range links {
		if links[i].Role == sdk.PermissionRead {
			continue
		}
		projectIDs = append(projectIDs, links[i].ProjectID)
	}
	if len(projectIDs) == 0 {
		return nil
	}

	// Compute organization for each project
	links, err = LoadLinksGroupProjectForProjectIDs(ctx, db, projectIDs, LoadLinkGroupProjectOptions.WithGroups)
	if err != nil {
		return err
	}
	mapProjectLinks := make(map[int64]sdk.GroupPermissions)
	for _, link := range links {
		if _, ok := mapProjectLinks[link.ProjectID]; !ok {
			mapProjectLinks[link.ProjectID] = nil
		}
		mapProjectLinks[link.ProjectID] = append(mapProjectLinks[link.ProjectID], sdk.GroupPermission{
			Permission: link.Role,
			Group:      link.Group,
		})
	}

	// For each project compute organization
	for k, gps := range mapProjectLinks {
		if _, err := gps.ComputeOrganization(); err != nil {
			return sdk.NewErrorFrom(err, "changing group organization conflict on project with id: %d", k)
		}
	}

	return nil
}
