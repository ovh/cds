package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// DeleteUserFromGroup remove user from group
func DeleteUserFromGroup(db gorp.SqlExecutor, groupID, userID int64) error {
	// Check if there are admin left
	var isAdm bool
	err := db.QueryRow(`SELECT group_admin FROM "group_user" WHERE group_id = $1 AND user_id = $2`, groupID, userID).Scan(&isAdm)
	if err != nil {
		return err
	}

	if isAdm {
		var nbAdm int
		err = db.QueryRow(`SELECT COUNT(id) FROM "group_user" WHERE group_id = $1 AND group_admin = true`, groupID).Scan(&nbAdm)
		if err != nil {
			return err
		}

		if nbAdm <= 1 {
			return sdk.ErrNotEnoughAdmin
		}
	}

	query := `DELETE FROM group_user WHERE group_id=$1 AND user_id=$2`
	_, err = db.Exec(query, groupID, userID)
	return err
}

// CheckUserInDefaultGroup insert user in default group
func CheckUserInDefaultGroup(ctx context.Context, db gorp.SqlExecutor, userID int64) error {
	if DefaultGroup == nil || DefaultGroup.ID == 0 {
		return nil
	}

	l, err := LoadLinkGroupUserForGroupIDAndUserID(ctx, db, DefaultGroup.ID, userID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}

	// If user is not in default group at it
	if l == nil {
		return InsertLinkGroupUser(db, &LinkGroupUser{
			GroupID: DefaultGroup.ID,
			UserID:  userID,
			Admin:   false,
		})
	}

	return nil
}

// LoadGroupByProject retrieves all groups related to project
func LoadGroupByProject(db gorp.SqlExecutor, project *sdk.Project) error {
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
