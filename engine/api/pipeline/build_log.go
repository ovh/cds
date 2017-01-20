package pipeline

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// InsertLog insert build log into database
func InsertLog(db database.Executer, actionBuildID int64, step string, value string) error {
	query := `INSERT INTO build_log (action_build_id, timestamp, step, value) VALUES ($1, $2, $3, $4)`

	_, err := db.Exec(query, actionBuildID, time.Now(), step, value)
	return err
}

// LoadLogs retrieves build logs from databse given an offset and a size
func LoadLogs(db *sql.DB, actionBuildID int64, tail int64, start int64) ([]sdk.Log, error) {
	query := `SELECT * FROM build_log WHERE action_build_id = $1`
	var logs []sdk.Log

	if start > 0 {
		query = fmt.Sprintf("%s AND id > %d", query, start)
	}

	query = fmt.Sprintf("%s ORDER BY id", query)
	if tail == 0 {
		tail = 5000
	}
	query = fmt.Sprintf("%s LIMIT %d", query, tail)

	rows, err := db.Query(query, actionBuildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var l sdk.Log
		err = rows.Scan(&l.ID, &l.ActionBuildID, &l.Timestamp, &l.Step, &l.Value)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}

	return logs, nil
}

// LoadPipelineActionBuildLogs Load log for the given pipeline action
func LoadPipelineActionBuildLogs(db *sql.DB, pipelineBuildID, pipelineActionID int64, offset int64) (sdk.BuildState, error) {
	buildLogResult := sdk.BuildState{}

	// load all build id for pipeline build
	query := `SELECT id, status FROM action_build WHERE pipeline_build_id = $1 AND pipeline_action_id=$2`
	rows, err := db.Query(query, pipelineBuildID, pipelineActionID)
	if err != nil {
		return buildLogResult, err
	}
	defer rows.Close()

	var actionBuilds []sdk.ActionBuild
	for rows.Next() {
		var b sdk.ActionBuild
		var sStatus string
		err = rows.Scan(&b.ID, &sStatus)
		b.Status = sdk.StatusFromString(sStatus)
		if err != nil {
			return buildLogResult, err
		}
		actionBuilds = append(actionBuilds, b)
	}

	pipelinelogs := []sdk.Log{}
	for _, build := range actionBuilds {

		logs, err := LoadLogs(db, build.ID, 0, offset)
		if err != nil {
			return buildLogResult, err
		}
		pipelinelogs = append(pipelinelogs, logs...)
	}

	buildLogResult.Logs = pipelinelogs
	if len(actionBuilds) == 1 {
		buildLogResult.Status = actionBuilds[0].Status
	}

	return buildLogResult, nil
}

// DeleteBuildLogs delete build log
func DeleteBuildLogs(db database.Executer, actionBuildID int64) error {
	query := `DELETE FROM build_log WHERE action_build_id = $1`
	_, err := db.Exec(query, actionBuildID)
	return err
}

// LoadPipelineBuildLogs Load pipeline build logs by pipeline ID
func LoadPipelineBuildLogs(db *sql.DB, pipelineBuildID int64, offset int64) ([]sdk.Log, error) {

	// load all build id for pipeline build
	query := `SELECT id FROM action_build WHERE pipeline_build_id = $1`
	rows, err := db.Query(query, pipelineBuildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actionBuildIDs []int
	for rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		actionBuildIDs = append(actionBuildIDs, id)
	}

	log.Debug("getBuildLogsHandler> ids: %v\n", actionBuildIDs)

	var pipelinelogs []sdk.Log
	for _, id := range actionBuildIDs {
		logs, err := LoadLogs(db, int64(id), 0, offset)
		if err != nil {
			return nil, err
		}
		pipelinelogs = append(pipelinelogs, logs...)
	}

	return pipelinelogs, nil
}
