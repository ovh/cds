package workflow

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func IsFavorite(db gorp.SqlExecutor, w *sdk.Workflow, uID string) (bool, error) {
	count, err := db.SelectInt("SELECT COUNT(1) FROM workflow_favorite WHERE authentified_user_id = $1 AND workflow_id = $2", uID, w.ID)
	if err != nil {
		return false, sdk.WithStack(err)
	}
	return count > 0, nil
}

// UserFavoriteWorkflowIDs returns the list of workflow ID
func UserFavoriteWorkflowIDs(db gorp.SqlExecutor, uID string) ([]int64, error) {
	var result []int64
	if _, err := db.Select(&result, "SELECT workflow_id FROM workflow_favorite WHERE authentified_user_id = $1", uID); err != nil {
		return nil, sdk.WithStack(err)
	}
	return result, nil
}

// LoadAllFavoritesNames returns all workflow for given project ids.
func LoadAllFavoritesNames(ctx context.Context, db gorp.SqlExecutor, user *sdk.AuthentifiedUser) ([]sdk.WorkflowName, error) {
	query := `SELECT workflow.*, project.projectkey
	FROM workflow
	JOIN project ON project.id = workflow.project_id
	JOIN workflow_favorite ON workflow.id = workflow_id
    WHERE authentified_user_id = $1`

	var result []sdk.WorkflowName // This struct is not registered as a gorpmapping entity so we can't use gorpmapping.Query
	_, err := db.Select(&result, query, user.ID)
	if err == sql.ErrNoRows {
		return result, nil
	}
	return result, sdk.WithStack(err)
}
