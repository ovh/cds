package user

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// InsertTimelineFilter inserts user timeline filter
func InsertTimelineFilter(db gorp.SqlExecutor, tf sdk.TimelineFilter, userID string) error {
	filterNullString, err := gorpmapping.JSONToNullString(tf)
	if err != nil {
		return sdk.WrapError(err, "Unable to insert filter")
	}
	if _, err := db.Exec("INSERT INTO user_timeline (authentified_user_id, filter) VALUES($1, $2)", userID, filterNullString); err != nil {
		return sdk.WrapError(err, "Unable to insert user timeline filter")
	}
	return nil
}

// UpdateTimelineFilter user timeline filter
func UpdateTimelineFilter(db gorp.SqlExecutor, timelineFilter sdk.TimelineFilter, userID string) error {
	filterJSON, err := gorpmapping.JSONToNullString(timelineFilter)
	if err != nil {
		return sdk.WrapError(err, "Unable to read json filter")
	}

	query := "UPDATE user_timeline SET filter=$1 WHERE authentified_user_id=$2"
	if _, err := db.Exec(query, filterJSON, userID); err != nil {
		return sdk.WrapError(err, "Unable to update filter")
	}
	return nil
}

// CountTimelineFilter count if user has a timeline filter
func CountTimelineFilter(db gorp.SqlExecutor, userID string) (int64, error) {
	return db.SelectInt("SELECT COUNT(*) from user_timeline WHERE authentified_user_id = $1", userID)
}

// Load user timeline filter
func LoadTimelineFilter(db gorp.SqlExecutor, userID string) (sdk.TimelineFilter, error) {
	var filter sdk.TimelineFilter
	var filterS sql.NullString
	query := "SELECT filter from user_timeline WHERE authentified_user_id = $1"
	err := db.QueryRow(query, userID).Scan(&filterS)
	if err != nil && err != sql.ErrNoRows {
		return filter, sdk.WrapError(err, "Unable to load timeline filter")
	}
	if err != nil && err == sql.ErrNoRows {
		filter = sdk.TimelineFilter{}
	}
	if err == nil {
		if err := gorpmapping.JSONNullString(filterS, &filter); err != nil {
			return filter, sdk.WrapError(err, "Unable to read filter")
		}
	}
	return filter, nil
}
