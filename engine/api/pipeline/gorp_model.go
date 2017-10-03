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

// PipelineAudit is a gorp wrapper around sdk.PipelineAudit
type PipelineAudit sdk.PipelineAudit

var Store cache.Store

//PostInsert is a DB Hook on PipelineBuildJob to store jobs and params as JSON in DB
func (p *PipelineBuildJob) PostInsert(s gorp.SqlExecutor) error {
	params, errParams := json.Marshal(p.Parameters)
	if errParams != nil {
		return errParams
	}
	job, errJob := json.Marshal(p.Job)
	if errJob != nil {
		return sdk.WrapError(errJob, "PostInsert> err on marshal p.Job")
	}
	spawn, errS := json.Marshal(p.SpawnInfos)
	if errS != nil {
		return sdk.WrapError(errS, "PostInsert> err on marshal p.SpawnInfos")
	}
	execGroups, errG := json.Marshal(p.ExecGroups)
	if errG != nil {
		return sdk.WrapError(errG, "PostInsert> err on marshal p.ExecGroups")
	}

	query := "update pipeline_build_job set parameters = $1, job = $2, spawninfos = $4, exec_groups = $5 where id = $3"
	if _, err := s.Exec(query, params, job, p.ID, spawn, execGroups); err != nil {
		return sdk.WrapError(err, "PostInsert> err on update sql")
	}
	return nil
}

//PostUpdate is a DB Hook on PipelineBuildJob to store JSON in DB
func (p *PipelineBuildJob) PostUpdate(s gorp.SqlExecutor) error {
	jobJSON, err := json.Marshal(p.Job)
	if err != nil {
		return sdk.WrapError(err, "PostUpdate> err on marshal p.Job")
	}

	paramsJSON, errP := json.Marshal(p.Parameters)
	if errP != nil {
		return sdk.WrapError(errP, "PostUpdate> err on marshal p.Parameters")
	}

	spawnJSON, errJ := json.Marshal(p.SpawnInfos)
	if errJ != nil {
		return sdk.WrapError(errJ, "PostUpdate> err on marshal p.SpawnInfos")
	}

	execGroupsJSON, errE := json.Marshal(p.ExecGroups)
	if errE != nil {
		return sdk.WrapError(errE, "PostUpdate> err on marshal p.ExecGroups")
	}

	// no need to update exec_groups, there are computed only at insert of pbj

	query := "update pipeline_build_job set job = $2, parameters = $3, spawninfos= $4, exec_groups= $5 where id = $1"
	if _, err := s.Exec(query, p.ID, jobJSON, paramsJSON, spawnJSON, execGroupsJSON); err != nil {
		return sdk.WrapError(err, "PostUpdate> err on update sql")
	}

	return nil
}

//PostGet is a DB Hook on PipelineBuildJob to get jobs and params as from JSON in DB
func (p *PipelineBuildJob) PostGet(s gorp.SqlExecutor) error {
	h := sdk.Hatchery{}
	if Store.Get(keyBookJob(p.ID), &h) {
		p.BookedBy = h
	}

	query := "SELECT job, parameters, spawninfos, exec_groups FROM pipeline_build_job WHERE id = $1"
	var params, job, spawn, execGroups []byte
	if err := s.QueryRow(query, p.ID).Scan(&job, &params, &spawn, &execGroups); err != nil {
		return sdk.WrapError(err, "PostGet> error on queryRow")
	}

	if err := json.Unmarshal(job, &p.Job); err != nil {
		return sdk.WrapError(err, "PostGet> error on unmarshal job")
	}
	if err := json.Unmarshal(params, &p.Parameters); err != nil {
		return sdk.WrapError(err, "PostGet> error on unmarshal params")
	}
	if err := json.Unmarshal(spawn, &p.SpawnInfos); err != nil {
		return sdk.WrapError(err, "PostGet> error on unmarshal spawnInfos")
	}
	if len(execGroups) > 0 {
		if err := json.Unmarshal(execGroups, &p.ExecGroups); err != nil {
			return sdk.WrapError(err, "PostGet> error on unmarshal exec_groups")
		}
	}

	p.QueuedSeconds = time.Now().Unix() - p.Queued.Unix()

	return nil
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(PipelineBuildJob{}, "pipeline_build_job", true, "id"),
		gorpmapping.New(Log{}, "pipeline_build_log", true, "id"),
		gorpmapping.New(PipelineAudit{}, "pipeline_audit", true, "id"),
	)
}
