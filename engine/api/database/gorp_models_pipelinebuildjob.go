package database

import (
	"encoding/json"

	"github.com/go-gorp/gorp"
)

//PostInsert is a DB Hook on PipelineBuildJob to store jobs and params as JSON in DB
func (p *PipelineBuildJob) PostInsert(s gorp.SqlExecutor) error {
	params, errParams := json.Marshal(p.Parameters)
	if errParams != nil {
		return errParams
	}
	job, errJob := json.Marshal(p.Job)
	if errJob != nil {
		return errJob
	}

	query := "update pipeline_build_job set parameters = $1, job = $2 where id = $3"
	if _, err := s.Exec(query, params, job, p.ID); err != nil {
		return err
	}
	return nil
}

//PostSelect is a DB Hook on PipelineBuildJob to get jobs and params as from JSON in DB
func (p *PipelineBuildJob) PostSelect(s gorp.SqlExecutor) error {
	if err := json.Unmarshal(p.JobJSON, &p.Job); err != nil {
		return err
	}
	if err := json.Unmarshal(p.ParametersJSON, &p.Parameters); err != nil {
		return err
	}
	return nil
}
