package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// DeleteUserFromGroup remove user from group
func DeleteUserFromGroup(ctx context.Context, db gorp.SqlExecutor, groupID int64, userID string) error {

	// Check if there are admin left
	grpLink, err := LoadLinkGroupUserForGroupIDAndUserID(ctx, db, groupID, userID)
	if err != nil {
		return err
	}

	if grpLink.Admin {
		var q = gorpmapping.NewQuery(`
			SELECT COUNT(id) 
			FROM "group_authentified_user" 
			WHERE group_id = $1 
			AND group_admin = true`).Args(groupID)
		nbAdmin, err := gorpmapping.GetInt(db, q)
		if err != nil {
			return err
		}

		if nbAdmin <= 1 {
			return sdk.WithStack(sdk.ErrNotEnoughAdmin)
		}
	}

	return DeleteLinkGroupUser(db, grpLink)
}

// CheckUserInDefaultGroup insert user in default group
func CheckUserInDefaultGroup(ctx context.Context, db gorp.SqlExecutor, userID string) error {
	if DefaultGroup == nil || DefaultGroup.ID == 0 || userID == "" {
		return nil
	}

	l, err := LoadLinkGroupUserForGroupIDAndUserID(ctx, db, DefaultGroup.ID, userID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}

	// If user is not in default group at it
	if l == nil {
		return InsertLinkGroupUser(ctx, db, &LinkGroupUser{
			GroupID:            DefaultGroup.ID,
			AuthentifiedUserID: userID,
			Admin:              false,
		})
	}

	return nil
}

// LoadGroupByProject retrieves all groups related to project
func LoadGroupByProject(db gorp.SqlExecutor, project *sdk.Project) error {

	// TODO sign this

	query := `
    SELECT "group".id, "group".name, project_group.role
    FROM "group"
	JOIN project_group ON project_group.group_id = "group".id
    WHERE project_group.project_id = $1
    ORDER BY "group".name ASC
  `
	rows, err := db.Query(query, project.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var group sdk.Group
		var perm int
		if err := rows.Scan(&group.ID, &group.Name, &perm); err != nil {
			return err
		}
		project.ProjectGroups = append(project.ProjectGroups, sdk.GroupPermission{
			Group:      group,
			Permission: perm,
		})
	}
	return nil
}
