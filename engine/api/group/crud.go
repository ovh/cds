package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// Create insert a new group in database and set user for given id as group admin.
func Create(ctx context.Context, db gorp.SqlExecutor, grp *sdk.Group, userID string) error {
	if err := Insert(ctx, db, grp); err != nil {
		return err
	}

	if err := InsertLinkGroupUser(ctx, db, &LinkGroupUser{
		GroupID:            grp.ID,
		AuthentifiedUserID: userID,
		Admin:              true,
	}); err != nil {
		return err
	}

	return nil
}

// Delete deletes group and dependencies.
func Delete(ctx context.Context, db gorp.SqlExecutor, g *sdk.Group) error {
	// To delete a group we need to check if it contains models, actions or templates
	// TODO

	// We can't delete a group if it's the last group on a project with RWX permissions
	linksForGroup, err := LoadLinksGroupProjectForGroupID(ctx, db, g.ID)
	if err != nil {
		return err
	}
	linksForProjects, err := LoadLinksGroupProjectForProjectIDs(ctx, db, linksForGroup.ToProjectIDs())
	if err != nil {
		return err
	}
	mapLinks := linksForProjects.ToMapByProjectID()
	for projectID, linksForProject := range mapLinks {
		var permissionOK bool
		for i := range linksForProject {
			if linksForProject[i].GroupID != g.ID && linksForProject[i].Role == sdk.PermissionReadWriteExecute {
				permissionOK = true
				break
			}
		}
		if !permissionOK {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot remove project as it's the last group with write permission on project %d", projectID)
		}
	}

	// Remove the group from database, this will also delete cascade group_user links
	return deleteDB(db, g)
}
