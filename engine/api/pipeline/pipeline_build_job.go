package pipeline

import (
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
	"encoding/json"
	"github.com/lib/pq"
	"database/sql"
)

// DeletePipelineBuildJob Delete all pipeline build job for the current pipeline build
func DeletePipelineBuildJob(db database.Executer, pipelineBuildID int64) error {
	query := "DELETE FROM pipeline_build_job WHERE pipeline_build_id = $1"
	_, err := db.Exec(query, pipelineBuildID)
	return err
}

// InsertPipelineBuildJob Insert a new job in the queue
func InsertPipelineBuildJob(db database.QueryExecuter, pbJob *sdk.PipelineBuildJob) error {
	params, errparams := json.Marshal(pbJob.Parameters)
	if errparams != nil {
		return errparams
	}
	job, errjob := json.Marshal(pbJob.Job)
	if errjob != nil {
		return errjob
	}
	query := `INSERT INTO pipeline_build_job
		(pipeline_build_id, parameters, job, status, queued, )
		VALUES ($1, p2, $3, $4, $5) RETURNING id`
	return db.QueryRow(query, pbJob.PipelineBuildID, params, job, pbJob.Status, pbJob.Queued).Scan(&pbJob.ID)
}

// GetPipelineBuildJob Get pipeline build job
func GetPipelineBuildJob(db database.QueryExecuter, id int64) (*sdk.PipelineBuildJob, error) {
	query := `
		SELECT id, job, parameters, status, queued, start, done, model, pipeline_build_id
		FROM pipeline_build_job
		WHERE id = $1
	`
	var pbJob sdk.PipelineBuildJob
	var start, done pq.NullTime
	var model sql.NullString
	var job, params, status string
	if err := db.QueryRow(query, id).Scan(&pbJob.ID, &job, &params, &status, &pbJob.Queued, &start, &done, &model, &pbJob.PipelineBuildID); err != nil {
		return nil, err
	}
	if start.Valid {
		pbJob.Start = start.Time
	}
	if done.Valid {
		pbJob.Done = done.Time
	}
	if model.Valid {
		pbJob.Model = model.String
	}

	if err := json.Unmarshal([]byte(job), &pbJob.Job); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(params), &pbJob.Parameters); err != nil {
		return nil, err
	}
	return &pbJob, nil
}