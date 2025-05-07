package bookmark

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func LoadAll(ctx context.Context, db gorp.SqlExecutor, userID string) ([]sdk.Bookmark, error) {
	query := `
		WITH results AS (
			(
				SELECT DISTINCT 'project' AS type, project.projectkey AS id, project.name AS label
				FROM project
				JOIN project_favorite ON project.id = project_favorite.project_id AND project_favorite.authentified_user_id = $1
			)
			UNION
			(
				SELECT 'workflow-legacy' AS type, CONCAT(project.projectkey, '/', workflow.name) AS id, workflow.name AS label
				FROM project
				JOIN workflow ON workflow.project_id = project.id
				JOIN workflow_favorite ON workflow.id = workflow_favorite.workflow_id AND workflow_favorite.authentified_user_id = $1
			)
			UNION
			(
				SELECT 'workflow' AS type, CONCAT(project.projectkey, '/', vcs_project.name, '/', project_repository.name, '/', entity_favorite.name) AS id, entity_favorite.name AS label
				FROM entity_favorite
				JOIN project_repository ON entity_favorite.project_repository_id = project_repository.id
				JOIN vcs_project ON project_repository.vcs_project_id = vcs_project.id
				JOIN project ON vcs_project.project_id = project.id
				WHERE entity_favorite.authentified_user_id = $1
			)
		)		
		SELECT *
		FROM results
		ORDER BY type ASC, label ASC
	`

	var data []sdk.Bookmark
	if _, err := db.Select(&data, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "cannot load bookmarks")
	}

	return data, nil
}

func InsertEntityFavorite(ctx context.Context, db gorp.SqlExecutor, authentifiedUserID, projectRepositoryID, entityType, entityName string) error {
	_, err := db.Exec(`
		INSERT INTO entity_favorite (authentified_user_id, project_repository_id, type, name) 
		VALUES ($1, $2, $3, $4)
	`, authentifiedUserID, projectRepositoryID, entityType, entityName)
	return sdk.WithStack(err)
}

func DeleteEntityFavorite(ctx context.Context, db gorp.SqlExecutor, authentifiedUserID, projectRepositoryID, entityType, entityName string) error {
	_, err := db.Exec(`
		DELETE FROM entity_favorite
		WHERE authentified_user_id = $1
			AND project_repository_id = $2
			AND type = $3
			AND name = $4
	`, authentifiedUserID, projectRepositoryID, entityType, entityName)
	return sdk.WithStack(err)
}

func InsertWorkflowLegacyFavorite(ctx context.Context, db gorp.SqlExecutor, authentifiedUserID string, workflowID int64) error {
	_, err := db.Exec(`
		INSERT INTO workflow_favorite (authentified_user_id, workflow_id) 
		VALUES ($1, $2)
	`, authentifiedUserID, workflowID)
	return sdk.WithStack(err)
}

func DeleteWorkflowLegacyFavorite(ctx context.Context, db gorp.SqlExecutor, authentifiedUserID string, workflowID int64) error {
	_, err := db.Exec(`
		DELETE FROM workflow_favorite
		WHERE authentified_user_id = $1 AND workflow_id = $2
	`, authentifiedUserID, workflowID)
	return sdk.WithStack(err)
}

func InsertProjectFavorite(ctx context.Context, db gorp.SqlExecutor, authentifiedUserID string, projectID int64) error {
	_, err := db.Exec(`
		INSERT INTO project_favorite (authentified_user_id, project_id) 
		VALUES ($1, $2)
	`, authentifiedUserID, projectID)
	return sdk.WithStack(err)
}

func DeleteProjectFavorite(ctx context.Context, db gorp.SqlExecutor, authentifiedUserID string, projectID int64) error {
	_, err := db.Exec(`
		DELETE FROM project_favorite
		WHERE authentified_user_id = $1 AND project_id = $2
	`, authentifiedUserID, projectID)
	return sdk.WithStack(err)
}
