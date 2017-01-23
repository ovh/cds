package pipeline

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// InsertLog insert build log into database
func InsertLog(db database.Executer, actionBuildID int64, step string, value string, pbID int64) error {
	query := `INSERT INTO build_log (action_build_id, timestamp, step, value, pipeline_build_id) VALUES ($1, $2, $3, $4, $5)`

	_, err := db.Exec(query, actionBuildID, time.Now(), step, value, pbID)
	return err
}

// LoadLogs retrieves build logs from databse given an offset and a size
func LoadLogs(db gorp.SqlExecutor, actionBuildID int64, tail int64, start int64) ([]sdk.Log, error) {
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
func LoadPipelineActionBuildLogs(db gorp.SqlExecutor, pipelineBuild *sdk.PipelineBuild, pipelineActionID int64, offset int64) (sdk.BuildState, error) {
	buildLogResult := sdk.BuildState{}

	// Found pipelien buid job from pipelineActionID
	var currentPbJob *sdk.PipelineBuildJob
	for _, s := range pipelineBuild.Stages {
		for _, pbJob := range s.PipelineBuildJobs {
			if pbJob.Job.PipelineActionID == pipelineActionID {
				currentPbJob = &pbJob
				break
			}
		}

	}

	if currentPbJob == nil {
		return buildLogResult, sdk.ErrNotFound
	}

	// Get the logs for the given pbJob
	var errLog error
	buildLogResult.Logs, errLog = LoadLogs(db, currentPbJob.ID, 0, offset)
	if errLog != nil {
		return buildLogResult, errLog
	}
	buildLogResult.Status = sdk.StatusFromString(currentPbJob.Status)

	return buildLogResult, nil
}

// DeleteBuildLogs delete build log
func DeleteBuildLogs(db database.Executer, actionBuildID int64) error {
	query := `DELETE FROM build_log WHERE action_build_id = $1`
	_, err := db.Exec(query, actionBuildID)
	return err
}

// LoadPipelineBuildLogs Load pipeline build logs by pipeline ID
func LoadPipelineBuildLogs(db gorp.SqlExecutor, pipelineBuildID int64, offset int64) ([]sdk.Log, error) {

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
