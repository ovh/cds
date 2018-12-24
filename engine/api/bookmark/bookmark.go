package bookmark

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadAll returns all bookmarks with icons and their description
func LoadAll(db gorp.SqlExecutor, u *sdk.User) ([]sdk.Bookmark, error) {
	var data []sdk.Bookmark
	query := `
	(
		SELECT DISTINCT
			project.projectkey AS key, project.name AS project_name, project.description, '' AS workflow_name, project.description, project.icon,
			true AS favorite,
			'project' AS type
		FROM project
		JOIN project_favorite ON project.id = project_favorite.project_id AND project_favorite.user_id = $1
		ORDER BY project.name
	)
	UNION
	(
		SELECT DISTINCT
			project.projectkey AS key, project.name AS project_name, workflow.description, workflow.name AS workflow_name, workflow.description, workflow.icon,
			true AS favorite,
			'workflow' AS type
		FROM project
		JOIN workflow ON workflow.project_id = project.id
		JOIN workflow_favorite ON workflow.id = workflow_favorite.workflow_id AND workflow_favorite.user_id = $1
		ORDER BY project.name
	)
	`
	if u == nil {
		u = &sdk.User{}
	}

	if _, err := db.Select(&data, query, u.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "cannot load bookmarks as admin")
	}

	return data, nil
}
