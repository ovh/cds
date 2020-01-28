package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// DeleteLinkGroupProject deletes the link between group and project and checks
// that group was not last with RWX permission.
func DeleteLinkGroupProject(db gorp.SqlExecutor, l *LinkGroupProject) error {
	query := `
    SELECT count(*)
    FROM project_group
    WHERE project_id = $1 AND id != $2 AND role = $3
  `
	nb, err := db.SelectInt(query, l.ProjectID, l.ID, sdk.PermissionReadWriteExecute)
	if err != nil {
		return sdk.WrapError(err, "cannot count link between project %d and group %d", l.ProjectID, l.GroupID)
	}
	if nb == 0 {
		return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot remove group from project as it's the last group with write permission on project")
	}

	return deleteDBLinkGroupProject(context.TODO(), db, l)
}

// UpdateLinkGroupProject updates group role for the given project.
func UpdateLinkGroupProject(db gorp.SqlExecutor, l *LinkGroupProject) error {
	// If downgrade of permission, checks that there is still a group with RWX permissions
	if l.Role < sdk.PermissionReadWriteExecute {
		query := `
      SELECT count(*)
      FROM project_group
      WHERE project_id = $1 AND id != $2 AND role = $3
    `
		nb, err := db.SelectInt(query, l.ProjectID, l.ID, 7)
		if err != nil {
			return sdk.WrapError(err, "cannot count link between project %d and group %d", l.ProjectID, l.GroupID)
		}
		if nb == 0 {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot downgrade group permission on project as it's the last group with write permission")
		}
	}

	return updateDBLinkGroupProject(context.TODO(), db, l)
}

// DeleteLinksGroupProjectForProjectID removes all links between group and project from database for given project id.
func DeleteLinksGroupProjectForProjectID(db gorp.SqlExecutor, projectID int64) error {
	_, err := db.Exec("DELETE FROM project_group WHERE project_id = $1", projectID)
	return sdk.WithStack(err)
}
