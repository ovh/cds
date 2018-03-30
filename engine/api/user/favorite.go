package user

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadFavorite return favorites from user given its user id
func LoadFavorite(db gorp.SqlExecutor, userID int64) (sdk.Favorite, error) {
	var favorite sdk.Favorite

	rowsP, errP := db.Query("SELECT project_id FROM project_favorite WHERE user_id = $1", userID)
	if errP != nil && errP != sql.ErrNoRows {
		return favorite, sdk.WrapError(errP, "LoadFavorites> Cannot load favorites for user id %d", userID)
	}
	defer rowsP.Close()

	for errP == nil && rowsP.Next() {
		var projectID int64
		if err := rowsP.Scan(&projectID); err != nil {
			return favorite, sdk.WrapError(err, "LoadFavorites> Cannot scan the row project for user id %d", userID)
		}

		if projectID != 0 {
			favorite.ProjectIDs = append(favorite.ProjectIDs, projectID)
		}
	}

	rowsW, errW := db.Query("SELECT project_id FROM project_favorite WHERE user_id = $1", userID)
	if errW != nil && errP != sql.ErrNoRows {
		return favorite, sdk.WrapError(errW, "LoadFavorites> Cannot load favorites for user id %d", userID)
	}
	defer rowsW.Close()

	for errW == nil && rowsW.Next() {
		var workflowID int64
		if err := rowsW.Scan(&workflowID); err != nil {
			return favorite, sdk.WrapError(err, "LoadFavorites> Cannot scan the row workflow for user id %d", userID)
		}

		if workflowID != 0 {
			favorite.WorkflowIDs = append(favorite.WorkflowIDs, workflowID)
		}
	}

	return favorite, nil
}
