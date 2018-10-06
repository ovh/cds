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

// QueueFilter contains all criterias used to fetch queue
type QueueFilter struct {
	ModelType    string
	RatioService *int
	GroupsID     []int64
	User         *sdk.User
	Rights       int
	Since        *time.Time
	Until        *time.Time
	Limit        *int
	Statuses     []string
}

// CountNodeJobRunQueue count all workflow_node_run_job accessible
func CountNodeJobRunQueue(ctx context.Context, db gorp.SqlExecutor, store cache.Store, filter QueueFilter) (sdk.WorkflowNodeJobRunCount, error) {
	c := sdk.WorkflowNodeJobRunCount{}

	queue, err := LoadNodeJobRunQueue(ctx, db, store, filter)
	if err != nil {
		return c, sdk.WrapError(err, "unable to load queue")
	}

	c.Count = int64(len(queue))
	if filter.Since != nil {
		c.Since = *filter.Since
	}
	if filter.Until != nil {
		c.Until = *filter.Until
	}
	return c, nil
}

// LoadNodeJobRunQueue load all workflow_node_run_job accessible
func LoadNodeJobRunQueue(ctx context.Context, db gorp.SqlExecutor, store cache.Store, filter QueueFilter) ([]sdk.WorkflowNodeJobRun, error) {
	ctx, end := observability.Span(ctx, "LoadNodeJobRunQueue")
	defer end()
	if filter.Since == nil {
		filter.Since = new(time.Time)
	}

	if filter.Until == nil {
		now := time.Now()
		filter.Until = &now
	}

	if len(filter.Statuses) == 0 {
		filter.Statuses = []string{sdk.StatusWaiting.String()}
	}

	containsService := []bool{true, false}
	if filter.RatioService != nil {
		if *filter.RatioService == 100 {
			containsService = []bool{true, true}
		} else if *filter.RatioService == 0 {
			containsService = []bool{false, false}
		}
	}

	modelTypes := sdk.AvailableWorkerModelType
	if filter.ModelType != "" {
		modelTypes = []string{filter.ModelType}
	}

	args := []interface{}{
		*filter.Since,                      // $1
		*filter.Until,                      // $2
		strings.Join(filter.Statuses, ","), // $3
		containsService[0],                 // $4
		containsService[1],                 // $5
		strings.Join(modelTypes, ","),      // $6
	}

	query := `select distinct workflow_node_run_job.*
	from workflow_node_run_job
	where workflow_node_run_job.queued >= $1
	and workflow_node_run_job.queued <= $2
	and workflow_node_run_job.status = ANY(string_to_array($3, ','))
	AND contains_service IN ($4, $5)
	AND (model_type is NULL OR model_type = '' OR model_type = ANY(string_to_array($6, ',')))
	ORDER BY workflow_node_run_job.queued ASC
	`

	if filter.User != nil && !filter.User.Admin {
		observability.Current(ctx, observability.Tag("isAdmin", false))
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
				project_group.group_id = ANY(string_to_array($7, ',')::int[])
			OR
				$8 = ANY(string_to_array($7, ',')::int[])
		)
		AND workflow_node_run_job.queued >= $1
		AND workflow_node_run_job.queued <= $2
		AND workflow_node_run_job.status = ANY(string_to_array($3, ','))
		AND contains_service IN ($4, $5)
		AND (model_type is NULL OR model_type = '' OR model_type = ANY(string_to_array($6, ',')))
		ORDER BY workflow_node_run_job.queued ASC
		`

		var groupID string
		for i, g := range filter.User.Groups {
			if i == 0 {
				groupID = fmt.Sprintf("%d", g.ID)
			} else {
				groupID += "," + fmt.Sprintf("%d", g.ID)
			}
		}
		args = append(args, groupID, group.SharedInfraGroup.ID)
	} else {
		observability.Current(ctx, observability.Tag("isAdmin", true))
	}

	if filter.Limit != nil && *filter.Limit > 0 {
		query += `
		LIMIT ` + strconv.Itoa(*filter.Limit)
	}
	isSharedInfraGroup := isSharedInfraGroup(filter.GroupsID)
	sqlJobs := []JobRun{}
	_, next := observability.Span(ctx, "LoadNodeJobRunQueue.select")
	if _, err := db.Select(&sqlJobs, query, args...); err != nil {
		next()
		return nil, sdk.WrapError(err, "Unable to load job runs (Select)")
	}
	next()

	ctx2, next2 := observability.Span(ctx, "LoadNodeJobRunQueue.sqlJobs")

	jobs := make([]sdk.WorkflowNodeJobRun, 0, len(sqlJobs))
	for i := range sqlJobs {
		_, next3 := observability.Span(ctx2, "LoadNodeJobRunQueue.loadHatcheryInfo")
		getHatcheryInfo(store, &sqlJobs[i])
		next3()
		jr, err := sqlJobs[i].WorkflowNodeRunJob()
		if err != nil {
			log.Error("LoadNodeJobRunQueue> WorkflowNodeRunJob error: %v", err)
		}

		var keepJobInQueue bool
		// a shared.infra group can see all jobs
		// a user (not a hatchery or worker) can see all jobs, even if jobs are only RO for him
		if isSharedInfraGroup || filter.Rights == permission.PermissionRead {
			keepJobInQueue = true
		} else {
			// if no shared.infra, we have to filter only executable jobs for worker or hatchery
			for _, g := range jr.ExecGroups {
				if sdk.IsInInt64Array(g.ID, filter.GroupsID) {
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
	next2()

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
	h := sdk.Service{}
	if store.Get(keyBookJob(j.ID), &h) {
		j.BookedBy = h
	}
}

// replaceWorkflowJobRunInQueue restart workflow node job
func replaceWorkflowJobRunInQueue(db gorp.SqlExecutor, wNodeJob sdk.WorkflowNodeJobRun) error {
	query := "UPDATE workflow_node_run_job SET status = $1, retry = $2, worker_id = NULL WHERE id = $3"
	if _, err := db.Exec(query, sdk.StatusWaiting.String(), wNodeJob.Retry+1, wNodeJob.ID); err != nil {
		return sdk.WrapError(err, "Unable to set workflow_node_run_job id %d with status %s", wNodeJob.ID, sdk.StatusWaiting.String())
	}

	query = "UPDATE worker SET status = $2, action_build_id = NULL where action_build_id = $1"
	if _, err := db.Exec(query, wNodeJob.ID, sdk.StatusDisabled); err != nil {
		return sdk.WrapError(err, "Unable to set workers")
	}

	return nil
}
