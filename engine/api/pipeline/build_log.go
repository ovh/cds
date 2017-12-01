package pipeline

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// UpdateLog Update a pipeline build step log
func UpdateLog(db gorp.SqlExecutor, l *sdk.Log) error {
	dbmodel := Log(*l)
	if _, err := db.Update(&dbmodel); err != nil {
		return err
	}
	return nil
}

// InsertLog insert build log into database
func InsertLog(db gorp.SqlExecutor, l *sdk.Log) error {
	dbmodel := Log(*l)
	if err := db.Insert(&dbmodel); err != nil {
		return err
	}
	*l = sdk.Log(dbmodel)
	return nil
}

// LoadStepLogs load log for the given pipeline build job at the given step
func LoadStepLogs(db gorp.SqlExecutor, pipJobID int64, stepOrder int64) (*sdk.Log, error) {
	var logGorp Log
	query := `
		SELECT *
		FROM pipeline_build_log
		WHERE pipeline_build_job_id = $1 AND step_order = $2
	`
	if err := db.SelectOne(&logGorp, query, pipJobID, stepOrder); err != nil {
		return nil, err
	}
	l := sdk.Log(logGorp)
	return &l, nil
}

// LoadLogs retrieves build logs from databse given an offset and a size
func LoadLogs(db gorp.SqlExecutor, pipelineJobID int64) ([]sdk.Log, error) {
	var logGorp []Log
	query := `
		SELECT *
		FROM pipeline_build_log
		WHERE pipeline_build_job_id = $1
		ORDER BY id
	`
	if _, err := db.Select(&logGorp, query, pipelineJobID); err != nil {
		return nil, err
	}
	var logs []sdk.Log
	for _, l := range logGorp {
		newLog := sdk.Log(l)
		logs = append(logs, newLog)
	}
	return logs, nil
}

// LoadPipelineBuildJobLogs Load log for the given pipeline action
func LoadPipelineBuildJobLogs(db gorp.SqlExecutor, pipelineBuild *sdk.PipelineBuild, pipelineActionID int64) (sdk.BuildState, error) {
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
	buildLogResult.Logs, errLog = LoadLogs(db, currentPbJob.ID)
	if errLog != nil {
		return buildLogResult, errLog
	}
	buildLogResult.Status = sdk.StatusFromString(currentPbJob.Status)

	return buildLogResult, nil
}

// DeleteBuildLogs delete build log
func DeleteBuildLogs(db gorp.SqlExecutor, pipJobID int64) error {
	query := `DELETE FROM pipeline_build_log WHERE pipeline_build_job_id = $1`
	_, err := db.Exec(query, pipJobID)
	return err
}

// DeleteBuildLogsByApplicationID Delete all log from the given build
func DeleteBuildLogsByApplicationID(db gorp.SqlExecutor, appID int64) error {
	query := `DELETE FROM pipeline_build_log WHERE pipeline_build_id IN (
				SELECT id from pipeline_build WHERE application_id = $1
			)`
	_, err := db.Exec(query, appID)
	return err
}

// DeleteBuildLogsByPipelineBuildID Delete all log from the given build
func DeleteBuildLogsByPipelineBuildID(db gorp.SqlExecutor, pipID int64) error {
	query := `DELETE FROM pipeline_build_log WHERE pipeline_build_id = $1`
	_, err := db.Exec(query, pipID)
	return err
}

//LoadPipelineStepBuildLogs loads build logs
func LoadPipelineStepBuildLogs(db gorp.SqlExecutor, pipelineBuild *sdk.PipelineBuild, pipelineActionID, stepOrder int64) (*sdk.BuildState, error) {
	var stepStatus string

	// Found pipeline buid job from pipelineActionID
	var currentPbJob *sdk.PipelineBuildJob
	for _, s := range pipelineBuild.Stages {
		for _, pbJob := range s.PipelineBuildJobs {
			if pbJob.Job.PipelineActionID == pipelineActionID {
				currentPbJob = &pbJob
				for _, step := range pbJob.Job.StepStatus {
					if step.StepOrder == int(stepOrder) {
						stepStatus = step.Status
						break
					}
				}
				break
			}
		}

	}

	if currentPbJob == nil {
		return nil, sdk.ErrNotFound
	}

	if stepStatus == "" {
		return nil, sdk.ErrNotFound
	}

	// Get the logs for the given pbJob
	logs, errLog := LoadStepLogs(db, currentPbJob.ID, stepOrder)
	if errLog != nil && errLog != sql.ErrNoRows {
		return nil, errLog
	}

	var buildLog sdk.Log
	if logs != nil {
		buildLog = *logs
	}

	result := &sdk.BuildState{
		Status:   sdk.StatusFromString(stepStatus),
		StepLogs: buildLog,
	}
	return result, nil
}

// LoadPipelineBuildLogs Load pipeline build logs by pipeline ID
func LoadPipelineBuildLogs(db gorp.SqlExecutor, pb *sdk.PipelineBuild) ([]sdk.Log, error) {

	var pipJobIDs []int64
	for _, s := range pb.Stages {
		for _, pbj := range s.PipelineBuildJobs {
			pipJobIDs = append(pipJobIDs, pbj.ID)
		}
	}

	log.Debug("getBuildLogsHandler> ids: %v\n", pipJobIDs)

	var pipelinelogs []sdk.Log
	for _, id := range pipJobIDs {
		logs, err := LoadLogs(db, int64(id))
		if err != nil {
			return nil, err
		}
		pipelinelogs = append(pipelinelogs, logs...)
	}

	return pipelinelogs, nil
}
