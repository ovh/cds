package pipeline

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

//AddBuildLog adds a build log
func AddBuildLog(db gorp.SqlExecutor, logs *sdk.Log) error {
	existingLogs, errLog := LoadStepLogs(db, logs.PipelineBuildJobID, logs.StepOrder)
	if errLog != nil && errLog != sql.ErrNoRows {
		return sdk.WrapError(errLog, "AddBuildLog> Cannot load existing logs")
	}

	if existingLogs == nil {
		if err := InsertLog(db, logs); err != nil {
			return sdk.WrapError(err, "Cannot insert log")
		}
	} else {
		existingLogs.Val += logs.Val
		existingLogs.LastModified = logs.LastModified
		existingLogs.Done = logs.Done
		if err := UpdateLog(db, existingLogs); err != nil {
			return sdk.WrapError(err, "Cannot update log")
		}
	}
	return nil
}
