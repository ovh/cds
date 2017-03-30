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
	spawn, errS := json.Marshal(p.SpawnInfos)
	if errS != nil {
		return errS
	}

	query := "update pipeline_build_job set parameters = $1, job = $2, spawninfos = $4 where id = $3"
	if _, err := s.Exec(query, params, job, p.ID, spawn); err != nil {
		return err
	}
	return nil
}

//PostUpdate is a DB Hook on PipelineBuildJob to store JSON in DB
func (p *PipelineBuildJob) PostUpdate(s gorp.SqlExecutor) error {
	jobJSON, err := json.Marshal(p.Job)
	if err != nil {
		return err
	}

	paramsJSON, errP := json.Marshal(p.Parameters)
	if errP != nil {
		return errP
	}

	spawnJSON, errJ := json.Marshal(p.SpawnInfos)
	if errJ != nil {
		return errJ
	}

	query := "update pipeline_build_job set job = $2, parameters = $3, spawninfos= $4 where id = $1"
	if _, err := s.Exec(query, p.ID, jobJSON, paramsJSON, spawnJSON); err != nil {
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

	query := "SELECT job, parameters, spawninfos FROM pipeline_build_job WHERE id = $1"
	var params, job, spawn []byte
	if err := s.QueryRow(query, p.ID).Scan(&job, &params, &spawn); err != nil {
		return err
	}

	if err := json.Unmarshal(job, &p.Job); err != nil {
		return err
	}
	if err := json.Unmarshal(params, &p.Parameters); err != nil {
		return err
	}
	if err := json.Unmarshal(spawn, &p.SpawnInfos); err != nil {
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
