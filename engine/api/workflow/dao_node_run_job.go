package workflow

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// loadPrepareGroup returns true if groupsID contains shareInfraGroup
// and list of groups, comma separated
func isSharedInfraGroup(groupsID []int64) bool {
	return sdk.IsInInt64Array(group.SharedInfraGroup.ID, groupsID)
}

// CountNodeJobRunQueue count all workflow_node_run_job accessible
func CountNodeJobRunQueue(db gorp.SqlExecutor, store cache.Store, groupsID []int64, usr *sdk.User, since *time.Time, until *time.Time, statuses ...string) (sdk.WorkflowNodeJobRunCount, error) {
	c := sdk.WorkflowNodeJobRunCount{}

	queue, err := LoadNodeJobRunQueue(db, store, permission.PermissionRead, groupsID, usr, since, until, nil, statuses...)
	if err != nil {
		return c, sdk.WrapError(err, "CountNodeJobRunQueue> unable to load queue")
	}

	c.Count = int64(len(queue))
	if since != nil {
		c.Since = *since
	}
	if until != nil {
		c.Until = *until
	}
	return c, nil
}

// LoadNodeJobRunQueue load all workflow_node_run_job accessible
func LoadNodeJobRunQueue(db gorp.SqlExecutor, store cache.Store, rights int, groupsID []int64, usr *sdk.User, since *time.Time, until *time.Time, limit *int, statuses ...string) ([]sdk.WorkflowNodeJobRun, error) {
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
	and workflow_node_run_job.status = ANY(string_to_array($3, ','))
	order by workflow_node_run_job.queued ASC`

	args := []interface{}{*since, *until, strings.Join(statuses, ",")}

	if usr != nil && !usr.Admin {
		query = `
		SELECT DISTINCT workflow_node_run_job.*
			FROM workflow_node_run_job
			JOIN workflow_node_run ON workflow_node_run.id = workflow_node_run_job.workflow_node_run_id
			JOIN workflow_run ON workflow_run.id = workflow_node_run.workflow_run_id
			JOIN project ON project.id = workflow_run.project_id
		WHERE project.id IN (
			SELECT project_group.project_id
			FROM project_group
			WHERE
				project_group.group_id = ANY(string_to_array($4, ',')::int[])
			OR
				$5 = ANY(string_to_array($4, ',')::int[])
		)
		AND workflow_node_run_job.queued >= $1
		AND workflow_node_run_job.queued <= $2
		AND workflow_node_run_job.status = ANY(string_to_array($3, ','))
		ORDER BY workflow_node_run_job.queued ASC
		`

		var groupID string
		for i, g := range usr.Groups {
			if i == 0 {
				groupID = fmt.Sprintf("%d", g.ID)
			} else {
				groupID += "," + fmt.Sprintf("%d", g.ID)
			}
		}
		args = append(args, groupID, group.SharedInfraGroup.ID)
	}

	if limit != nil && *limit > 0 {
		query += `
		LIMIT ` + strconv.Itoa(*limit)
	}

	isSharedInfraGroup := isSharedInfraGroup(groupsID)
	sqlJobs := []JobRun{}
	if _, err := db.Select(&sqlJobs, query, args...); err != nil {
		return nil, sdk.WrapError(err, "workflow.LoadNodeJobRun> Unable to load job runs (Select)")
	}

	jobs := make([]sdk.WorkflowNodeJobRun, 0, len(sqlJobs))
	for i := range sqlJobs {
		getHatcheryInfo(store, &sqlJobs[i])
		jr, err := sqlJobs[i].WorkflowNodeRunJob()
		if err != nil {
			log.Error("LoadNodeJobRunQueue> WorkflowNodeRunJob error: %v", err)
		}

		var keepJobInQueue bool
		// a shared.infra group can see all jobs
		// a user (not a hatchery or worker) can see all jobs, even if jobs are only RO for him
		if isSharedInfraGroup || rights == permission.PermissionRead {
			keepJobInQueue = true
		} else {
			// if no shared.infra, we have to filter only executable jobs for worker or hatchery
			for _, g := range jr.ExecGroups {
				if sdk.IsInInt64Array(g.ID, groupsID) {
					keepJobInQueue = true
					break
				}
			}
		}

		if !keepJobInQueue {
			continue
		}

		jobs = append(jobs, jr)
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
	jr, err := j.WorkflowNodeRunJob()
	if err != nil {
		return nil, err
	}
	return &jr, nil
}

//LoadAndLockNodeJobRunWait load for update a NodeJobRun given its ID
func LoadAndLockNodeJobRunWait(db gorp.SqlExecutor, store cache.Store, id int64) (*sdk.WorkflowNodeJobRun, error) {
	j := JobRun{}
	query := `select workflow_node_run_job.* from workflow_node_run_job where id = $1 for update`
	if err := db.SelectOne(&j, query, id); err != nil {
		return nil, err
	}
	getHatcheryInfo(store, &j)
	jr, err := j.WorkflowNodeRunJob()
	if err != nil {
		return nil, err
	}
	return &jr, nil
}

//LoadAndLockNodeJobRunNoWait load for update a NodeJobRun given its ID
func LoadAndLockNodeJobRunNoWait(ctx context.Context, db gorp.SqlExecutor, store cache.Store, id int64) (*sdk.WorkflowNodeJobRun, error) {
	var end func()
	_, end = observability.Span(ctx, "workflow.LoadAndLockNodeJobRunNoWait")
	defer end()

	j := JobRun{}
	query := `select workflow_node_run_job.* from workflow_node_run_job where id = $1 for update nowait`
	if err := db.SelectOne(&j, query, id); err != nil {
		return nil, err
	}
	getHatcheryInfo(store, &j)
	jr, err := j.WorkflowNodeRunJob()
	if err != nil {
		return nil, err
	}
	return &jr, nil
}

func insertWorkflowNodeJobRun(db gorp.SqlExecutor, j *sdk.WorkflowNodeJobRun) error {
	dbj := new(JobRun)
	err := dbj.ToJobRun(j)
	if err != nil {
		return err
	}
	if err := db.Insert(dbj); err != nil {
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
func UpdateNodeJobRun(ctx context.Context, db gorp.SqlExecutor, j *sdk.WorkflowNodeJobRun) error {
	var end func()
	_, end = observability.Span(ctx, "workflow.UpdateNodeJobRun")
	defer end()

	dbj := new(JobRun)
	err := dbj.ToJobRun(j)
	if err != nil {
		return err
	}
	if _, err := db.Update(dbj); err != nil {
		return err
	}
	return nil
}

func keyBookJob(id int64) string {
	return cache.Key("book", "job", strconv.FormatInt(id, 10))
}

func getHatcheryInfo(store cache.Store, j *JobRun) {
	h := sdk.Hatchery{}
	if store.Get(keyBookJob(j.ID), &h) {
		j.BookedBy = h
	}
}

// replaceWorkflowJobRunInQueue restart workflow node job
func replaceWorkflowJobRunInQueue(db gorp.SqlExecutor, wNodeJob sdk.WorkflowNodeJobRun) error {
	query := "UPDATE workflow_node_run_job SET status = $1, retry = $2, worker_id = NULL WHERE id = $3"
	if _, err := db.Exec(query, sdk.StatusWaiting.String(), wNodeJob.Retry+1, wNodeJob.ID); err != nil {
		return sdk.WrapError(err, "replaceWorkflowJobRunInQueue> Unable to set workflow_node_run_job id %d with status %s", wNodeJob.ID, sdk.StatusWaiting.String())
	}

	query = "UPDATE worker SET status = $2, action_build_id = NULL where action_build_id = $1"
	if _, err := db.Exec(query, wNodeJob.ID, sdk.StatusDisabled); err != nil {
		return sdk.WrapError(err, "replaceWorkflowJobRunInQueue> Unable to set workers")
	}

	return nil
}
