package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
)

// loadPrepareGroup returns true if groupsID contains shareInfraGroup
// and list of groups, comma separated
func isSharedInfraGroup(groupsID []int64) bool {
	return sdk.IsInInt64Array(group.SharedInfraGroup.ID, groupsID)
}

// CountNodeJobRunQueue count all workflow_node_run_job accessible
func CountNodeJobRunQueue(db gorp.SqlExecutor, store cache.Store, groupsID []int64, since *time.Time, until *time.Time, statuses ...string) (sdk.WorkflowNodeJobRunCount, error) {
	c := sdk.WorkflowNodeJobRunCount{}

	queue, err := LoadNodeJobRunQueue(db, store, permission.PermissionRead, groupsID, since, until, statuses...)
	if err != nil {
		return c, sdk.WrapError(err, "CountNodeJobRunQueue> unable to load queue")
	}

	c.Count = int64(len(queue))
	c.Since = *since
	c.Until = *until
	return c, nil
}

// LoadNodeJobRunQueue load all workflow_node_run_job accessible
func LoadNodeJobRunQueue(db gorp.SqlExecutor, store cache.Store, rights int, groupsID []int64, since *time.Time, until *time.Time, statuses ...string) ([]sdk.WorkflowNodeJobRun, error) {
	if since == nil {
		since = new(time.Time)
	}

	if until == nil {
		now := time.Now()
		until = &now
	}

	if len(statuses) == 0 {
		statuses = []string{sdk.StatusWaiting.String()}
	}

	query := `select distinct workflow_node_run_job.* 
	from workflow_node_run_job
	where workflow_node_run_job.queued >= $1
	and workflow_node_run_job.queued <= $2
	and workflow_node_run_job.status = ANY(string_to_array($3, ','))`

	isSharedInfraGroup := isSharedInfraGroup(groupsID)
	sqlJobs := []JobRun{}
	if _, err := db.Select(&sqlJobs, query, *since, *until, strings.Join(statuses, ",")); err != nil {
		return nil, sdk.WrapError(err, "workflow.LoadNodeJobRun> Unable to load job runs (Select)")
	}

	jobs := []sdk.WorkflowNodeJobRun{}
	for i := range sqlJobs {
		if err := sqlJobs[i].PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "workflow.LoadNodeJobRun> Unable to load job runs (PostGet)")
		}

		var keepJobInQueue bool
		// a shared.infra group can see all jobs
		// a user (not a hatchery or worker) can see all jobs, even if jobs are only RO for him
		if isSharedInfraGroup || rights == permission.PermissionRead {
			keepJobInQueue = true
		} else {
			// if no shared.infra, we have to filter only executable jobs for worker or hatchery
			for _, g := range sqlJobs[i].ExecGroups {
				if sdk.IsInInt64Array(g.ID, groupsID) {
					keepJobInQueue = true
					break
				}
			}
		}

		if !keepJobInQueue {
			continue
		}
		getHatcheryInfo(store, &sqlJobs[i])
		jobs = append(jobs, sdk.WorkflowNodeJobRun(sqlJobs[i]))
	}

	return jobs, nil
}

// LoadNodeJobRunIDByNodeRunID Load node run job id by node run id
func LoadNodeJobRunIDByNodeRunID(db gorp.SqlExecutor, runNodeID int64) ([]int64, error) {
	query := `SELECT workflow_node_run_job.id FROM workflow_node_run_job WHERE workflow_node_run_id = $1`
	rows, err := db.Query(query, runNodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

//LoadNodeJobRun load a NodeJobRun given its ID
func LoadNodeJobRun(db gorp.SqlExecutor, store cache.Store, id int64) (*sdk.WorkflowNodeJobRun, error) {
	j := JobRun{}
	query := `select workflow_node_run_job.* from workflow_node_run_job where id = $1`
	if err := db.SelectOne(&j, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrWorkflowNodeRunJobNotFound
		}
		return nil, err
	}
	if store != nil {
		getHatcheryInfo(store, &j)
	}
	job := sdk.WorkflowNodeJobRun(j)
	return &job, nil
}

//LoadAndLockNodeJobRunWait load for update a NodeJobRun given its ID
func LoadAndLockNodeJobRunWait(db gorp.SqlExecutor, store cache.Store, id int64) (*sdk.WorkflowNodeJobRun, error) {
	j := JobRun{}
	query := `select workflow_node_run_job.* from workflow_node_run_job where id = $1 for update`
	if err := db.SelectOne(&j, query, id); err != nil {
		return nil, err
	}
	getHatcheryInfo(store, &j)
	job := sdk.WorkflowNodeJobRun(j)
	return &job, nil
}

//LoadAndLockNodeJobRunNoWait load for update a NodeJobRun given its ID
func LoadAndLockNodeJobRunNoWait(db gorp.SqlExecutor, store cache.Store, id int64) (*sdk.WorkflowNodeJobRun, error) {
	j := JobRun{}
	query := `select workflow_node_run_job.* from workflow_node_run_job where id = $1 for update nowait`
	if err := db.SelectOne(&j, query, id); err != nil {
		return nil, err
	}
	getHatcheryInfo(store, &j)
	job := sdk.WorkflowNodeJobRun(j)
	return &job, nil
}

func insertWorkflowNodeJobRun(db gorp.SqlExecutor, j *sdk.WorkflowNodeJobRun) error {
	dbj := JobRun(*j)
	if err := db.Insert(&dbj); err != nil {
		return err
	}
	j.ID = dbj.ID
	return nil
}

//DeleteNodeJobRuns deletes all workflow_node_run_job for a given workflow_node_run
func DeleteNodeJobRuns(db gorp.SqlExecutor, nodeID int64) error {
	query := `delete from workflow_node_run_job where workflow_node_run_id = $1`
	_, err := db.Exec(query, nodeID)
	return err
}

//UpdateNodeJobRun updates a workflow_node_run_job
func UpdateNodeJobRun(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, j *sdk.WorkflowNodeJobRun) error {
	dbj := JobRun(*j)
	if _, err := db.Update(&dbj); err != nil {
		return err
	}
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

	paramsJSON, errP := json.Marshal(j.Parameters)
	if errP != nil {
		return errP
	}

	execGroupsJSON, errG := json.Marshal(j.ExecGroups)
	if errG != nil {
		return sdk.WrapError(errG, "PostUpdate> err on marshal j.ExecGroups")
	}

	query := "update workflow_node_run_job set job = $2, variables = $3, exec_groups = $4 where id = $1"
	if n, err := s.Exec(query, j.ID, jobJSON, paramsJSON, execGroupsJSON); err != nil {
		return err
	} else if n, _ := n.RowsAffected(); n == 0 {
		return fmt.Errorf("Unable to update workflow_node_run_job id = %d", j.ID)
	}

	return nil
}

func getHatcheryInfo(store cache.Store, j *JobRun) {
	h := sdk.Hatchery{}
	if store.Get(keyBookJob(j.ID), &h) {
		j.BookedBy = h
	}
}

// PostGet is a db hook on workflow_node_run_job
func (j *JobRun) PostGet(s gorp.SqlExecutor) error {
	query := "SELECT job, variables, exec_groups FROM workflow_node_run_job WHERE id = $1"
	var params, job, execGroups []byte
	if err := s.QueryRow(query, j.ID).Scan(&job, &params, &execGroups); err != nil {
		return sdk.WrapError(err, "PostGet> s.QueryRow id:%d", j.ID)
	}
	if err := json.Unmarshal(job, &j.Job); err != nil {
		return sdk.WrapError(err, "PostGet> json.Unmarshal job")
	}
	if err := json.Unmarshal(params, &j.Parameters); err != nil {
		return sdk.WrapError(err, "PostGet> json.Unmarshal params")
	}

	if len(execGroups) > 0 {
		if err := json.Unmarshal(execGroups, &j.ExecGroups); err != nil {
			return sdk.WrapError(err, "PostGet> error on unmarshal exec_groups")
		}
	}

	rows, err := s.Query("SELECT DISTINCT UNNEST(spawn_attempts) FROM workflow_node_run_job WHERE id = $1", j.ID)
	if err != nil && err != sql.ErrNoRows {
		return sdk.WrapError(err, "PostGet> cannot get spawn_attempts")
	}

	var hID int64
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&hID); err != nil {
			return sdk.WrapError(err, "PostGet> cannot scan spawn_attempts")
		}
		j.SpawnAttempts = append(j.SpawnAttempts, hID)
	}

	j.QueuedSeconds = time.Now().Unix() - j.Queued.Unix()
	return nil
}

// replaceWorkflowJobRunInQueue restart workflow node job
func replaceWorkflowJobRunInQueue(db gorp.SqlExecutor, wNodeJob sdk.WorkflowNodeJobRun) error {
	query := "UPDATE workflow_node_run_job SET status = $1, retry = $2 WHERE id = $3"
	if _, err := db.Exec(query, sdk.StatusWaiting.String(), wNodeJob.Retry+1, wNodeJob.ID); err != nil {
		return sdk.WrapError(err, "replaceWorkflowJobRunInQueue> Unable to set workflow_node_run_job id %d with status %s", wNodeJob.ID, sdk.StatusWaiting.String())
	}
	return nil
}
