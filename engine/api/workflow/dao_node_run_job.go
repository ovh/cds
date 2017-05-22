package workflow

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func insertWorkflowNodeJobRun(db gorp.SqlExecutor, j *sdk.WorkflowNodeJobRun) error {
	dbj := JobRun(*j)
	if err := db.Insert(&dbj); err != nil {
		return err
	}
	j.ID = dbj.ID

	log.Debug("insertWorkflowNodeJobRun> %d", j.ID)

	return nil
}

func keyBookJob(id int64) string {
	return cache.Key("book", "job", strconv.FormatInt(id, 10))
}

// PostInsert is a db hook on workflow_node_run_job
func (j *JobRun) PostInsert(s gorp.SqlExecutor) error {
	return j.PostUpdate(s)
}

// PostUpdate is a db hook on workflow_node_run_job
func (j *JobRun) PostUpdate(s gorp.SqlExecutor) error {
	jobJSON, err := json.Marshal(j.Job)
	if err != nil {
		return err
	}

	paramsJSON, errP := json.Marshal(j.Variables)
	if errP != nil {
		return errP
	}

	spawnJSON, errJ := json.Marshal(j.SpawnInfos)
	if errJ != nil {
		return errJ
	}

	query := "update workflow_node_run_job set job = $2, variables = $3, spawninfos= $4 where id = $1"
	if _, err := s.Exec(query, j.ID, jobJSON, paramsJSON, spawnJSON); err != nil {
		return err
	}

	return nil
}

// PostGet is a db hook on workflow_node_run_job
func (j *JobRun) PostGet(s gorp.SqlExecutor) error {
	h := sdk.Hatchery{}
	if cache.Get(keyBookJob(j.ID), &h) {
		j.BookedBy = h
	}

	query := "SELECT job, variables, spawninfos FROM workflow_node_run_job WHERE id = $1"
	var params, job, spawn []byte
	if err := s.QueryRow(query, j.ID).Scan(&job, &params, &spawn); err != nil {
		return err
	}

	if err := json.Unmarshal(job, &j.Job); err != nil {
		return err
	}
	if err := json.Unmarshal(params, &j.Variables); err != nil {
		return err
	}
	if err := json.Unmarshal(spawn, &j.SpawnInfos); err != nil {
		return err
	}

	j.QueuedSeconds = time.Now().Unix() - j.Queued.Unix()

	return nil
}
