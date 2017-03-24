package pipeline

import (
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// PipelineBuildJob is a gorp wrapper around sdk.PipelineBuildJob
type PipelineBuildJob sdk.PipelineBuildJob

// Log is a gorp wrapper around sdk.Log
type Log sdk.Log

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

//PostGet is a DB Hook on PipelineBuildJob to get jobs and params as from JSON in DB
func (p *PipelineBuildJob) PostGet(s gorp.SqlExecutor) error {
	h := sdk.Hatchery{}
	if cache.Get(keyBookJob(p.ID), &h) {
		p.BookedBy = h
	}

	if err := json.Unmarshal(p.JobJSON, &p.Job); err != nil {
		return err
	}
	if err := json.Unmarshal(p.ParametersJSON, &p.Parameters); err != nil {
		return err
	}
	if err := json.Unmarshal(p.SpawnInfosJSON, &p.SpawnInfos); err != nil {
		return err
	}

	p.QueuedSeconds = time.Now().Unix() - p.Queued.Unix()

	return nil
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(PipelineBuildJob{}, "pipeline_build_job", true, "id"),
		gorpmapping.New(Log{}, "pipeline_build_log", true, "id"),
	)
}
