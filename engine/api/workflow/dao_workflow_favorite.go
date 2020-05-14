package workflow

import (
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
