package workflow

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

// QueueFilter contains all criteria used to fetch queue
type QueueFilter struct {
	ModelType []string
	Rights    int
	Since     *time.Time
	Until     *time.Time
	Limit     *int
	Statuses  []string
	Regions   []string
}

func NewQueueFilter() QueueFilter {
	now := time.Now()
	return QueueFilter{
		ModelType: sdk.AvailableWorkerModelType,
		Rights:    sdk.PermissionRead,
		Since:     new(time.Time),
		Until:     &now,
		Statuses:  []string{sdk.StatusWaiting},
	}
}

// CountNodeJobRunQueue count all workflow_node_run_job accessible
func CountNodeJobRunQueue(ctx context.Context, db gorp.SqlExecutor, store cache.Store, filter QueueFilter) (sdk.WorkflowNodeJobRunCount, error) {
	var c sdk.WorkflowNodeJobRunCount
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

func CountNodeJobRunQueueByGroupIDs(ctx context.Context, db gorp.SqlExecutor, store cache.Store, filter QueueFilter, groupIDs []int64) (sdk.WorkflowNodeJobRunCount, error) {
	var c sdk.WorkflowNodeJobRunCount
	queue, err := LoadNodeJobRunQueueByGroupIDs(ctx, db, store, filter, groupIDs)
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
	ctx, end := telemetry.Span(ctx, "workflow.LoadNodeJobRunQueue")
	defer end()

	query := gorpmapping.NewQuery(`select distinct workflow_node_run_job.*
	from workflow_node_run_job
	where workflow_node_run_job.queued >= $1
	and workflow_node_run_job.queued <= $2
	and workflow_node_run_job.status = ANY($3)
	AND (model_type is NULL OR model_type = '' OR model_type = ANY($4))
  AND (
    workflow_node_run_job.region = ANY($5)
    OR
    (workflow_node_run_job.region is NULL AND '' = ANY($5))
    OR
    array_length($5, 1) is NULL
  )
	ORDER BY workflow_node_run_job.queued ASC
	`).Args(
		*filter.Since,                    // $1
		*filter.Until,                    // $2
		pq.StringArray(filter.Statuses),  // $3
		pq.StringArray(filter.ModelType), // $4
		pq.StringArray(filter.Regions),   // $5
	)

	return loadNodeJobRunQueue(ctx, db, store, query, filter.Limit)
}

// LoadNodeJobRunQueueByGroupIDs load all workflow_node_run_job accessible
func LoadNodeJobRunQueueByGroupIDs(ctx context.Context, db gorp.SqlExecutor, store cache.Store, filter QueueFilter, groupIDs []int64) ([]sdk.WorkflowNodeJobRun, error) {
	ctx, end := telemetry.Span(ctx, "workflow.LoadNodeJobRunQueueByGroups")
	defer end()

	query := gorpmapping.NewQuery(`
	-- Parameters:
	--  $1: Queue since
	--  $2: Queue until
	--  $3: List of status
	--  $4: List of model types
	--  $5: Comman separated list of groups ID
	--  $6: shared infra group ID
	--  $7: minimum level of permission
    --  $8: List of regions
	WITH workflow_id_with_permissions AS (
		SELECT workflow_perm.workflow_id,
			CASE WHEN $6 = ANY(string_to_array($5, ',')::int[]) THEN 7
				 ELSE max(workflow_perm.role)
			END as "role"
		FROM workflow_perm
		JOIN project_group ON project_group.id = workflow_perm.project_group_id
		WHERE
			project_group.group_id = ANY(string_to_array($5, ',')::int[])
		OR
			$6 = ANY(string_to_array($5, ',')::int[])
		GROUP BY workflow_perm.workflow_id
	), workflow_node_run_job_exec_groups AS (
		SELECT id, jsonb_array_elements_text(exec_groups)::jsonb->'id' AS exec_group_id
		FROM workflow_node_run_job
	), workflow_node_run_job_matching_exec_groups AS (
		SELECT id
		FROM workflow_node_run_job_exec_groups
		WHERE exec_group_id::text = ANY(string_to_array($5, ','))
	)
	SELECT DISTINCT workflow_node_run_job.*
	FROM workflow_node_run_job
	JOIN workflow_node_run ON workflow_node_run.id = workflow_node_run_job.workflow_node_run_id
	JOIN workflow_run ON workflow_run.id = workflow_node_run.workflow_run_id
	JOIN workflow ON workflow.id = workflow_run.workflow_id
	WHERE workflow.id IN (
		SELECT workflow_id
		FROM workflow_id_with_permissions
		WHERE role >= $7
	)
	AND workflow_node_run_job.id IN (
		SELECT id
		FROM workflow_node_run_job_matching_exec_groups
	)
	AND workflow_node_run_job.queued >= $1
	AND workflow_node_run_job.queued <= $2
	AND workflow_node_run_job.status = ANY($3)
	AND (
		workflow_node_run_job.model_type is NULL
		OR
		model_type = '' OR model_type = ANY($4)
	)
  AND (
    workflow_node_run_job.region = ANY($8)
    OR
    (workflow_node_run_job.region is NULL AND '' = ANY($8))
    OR
    array_length($8, 1) is NULL
  )
	ORDER BY workflow_node_run_job.queued ASC
	`).Args(
		*filter.Since,                          // $1
		*filter.Until,                          // $2
		pq.StringArray(filter.Statuses),        // $3
		pq.StringArray(filter.ModelType),       // $4
		gorpmapping.IDsToQueryString(groupIDs), // $5
		group.SharedInfraGroup.ID,              // $6
		filter.Rights,                          // $7
		pq.StringArray(filter.Regions),         // $8
	)
	return loadNodeJobRunQueue(ctx, db, store, query, filter.Limit)
}

func loadNodeJobRunQueue(ctx context.Context, db gorp.SqlExecutor, store cache.Store, query gorpmapping.Query, limit *int) ([]sdk.WorkflowNodeJobRun, error) {
	ctx, end := telemetry.Span(ctx, "workflow.loadNodeJobRunQueue")
	defer end()

	if limit != nil && *limit > 0 {
		query = query.Limit(*limit)
	}

	var sqlJobs []JobRun

	if err := gorpmapping.GetAll(ctx, db, query, &sqlJobs); err != nil {
		return nil, sdk.WrapError(err, "Unable to load job runs (Select)")
	}

	jobs := make([]sdk.WorkflowNodeJobRun, 0, len(sqlJobs))
	for i := range sqlJobs {
		getHatcheryInfo(ctx, store, &sqlJobs[i])
		jr, err := sqlJobs[i].WorkflowNodeRunJob()
		if err != nil {
			log.Error(ctx, "LoadNodeJobRunQueue> WorkflowNodeRunJob error: %v", err)
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
func LoadNodeJobRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, id int64) (*sdk.WorkflowNodeJobRun, error) {
	j := JobRun{}
	query := `select workflow_node_run_job.* from workflow_node_run_job where id = $1`
	if err := db.SelectOne(&j, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrWorkflowNodeRunJobNotFound)
		}
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == "55P03" {
			return nil, sdk.WithStack(sdk.ErrJobLocked)
		}
		return nil, sdk.WithStack(err)
	}
	if store != nil {
		getHatcheryInfo(ctx, store, &j)
	}
	jr, err := j.WorkflowNodeRunJob()
	if err != nil {
		return nil, err
	}
	return &jr, nil
}

//LoadDeadNodeJobRun load a NodeJobRun which is Building but without worker
func LoadDeadNodeJobRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store) ([]sdk.WorkflowNodeJobRun, error) {
	var deadJobsDB []JobRun
	query := `SELECT workflow_node_run_job.* FROM workflow_node_run_job WHERE worker_id IS NULL`
	if _, err := db.Select(&deadJobsDB, query); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	deadJobs := make([]sdk.WorkflowNodeJobRun, len(deadJobsDB))
	for i, deadJob := range deadJobsDB {
		if store != nil {
			getHatcheryInfo(ctx, store, &deadJob)
		}

		jr, err := deadJob.WorkflowNodeRunJob()
		if err != nil {
			return nil, err
		}
		deadJobs[i] = jr
	}

	return deadJobs, nil
}

//LoadAndLockNodeJobRunWait load for update a NodeJobRun given its ID
func LoadAndLockNodeJobRunWait(ctx context.Context, db gorp.SqlExecutor, store cache.Store, id int64) (*sdk.WorkflowNodeJobRun, error) {
	j := JobRun{}
	query := `select workflow_node_run_job.* from workflow_node_run_job where id = $1 for update`
	if err := db.SelectOne(&j, query, id); err != nil {
		return nil, err
	}
	getHatcheryInfo(ctx, store, &j)
	jr, err := j.WorkflowNodeRunJob()
	if err != nil {
		return nil, err
	}
	return &jr, nil
}

//LoadAndLockNodeJobRunSkipLocked load for update a NodeJobRun given its ID
func LoadAndLockNodeJobRunSkipLocked(ctx context.Context, db gorp.SqlExecutor, store cache.Store, id int64) (*sdk.WorkflowNodeJobRun, error) {
	var end func()
	_, end = telemetry.Span(ctx, "workflow.LoadAndLockNodeJobRunSkipLocked")
	defer end()

	j := JobRun{}
	query := `select workflow_node_run_job.* from workflow_node_run_job where id = $1 for update SKIP LOCKED`
	if err := db.SelectOne(&j, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrLocked)
		}
		return nil, err
	}
	getHatcheryInfo(ctx, store, &j)
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

// DeleteNodeJobRun deletes the given workflow_node_run_job
func DeleteNodeJobRun(db gorp.SqlExecutor, nodeRunJob int64) error {
	query := `delete from workflow_node_run_job where id = $1`
	_, err := db.Exec(query, nodeRunJob)
	return err
}

//UpdateNodeJobRun updates a workflow_node_run_job
func UpdateNodeJobRun(ctx context.Context, db gorp.SqlExecutor, j *sdk.WorkflowNodeJobRun) error {
	var end func()
	_, end = telemetry.Span(ctx, "workflow.UpdateNodeJobRun")
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

func getHatcheryInfo(ctx context.Context, store cache.Store, j *JobRun) {
	h := sdk.Service{}
	k := keyBookJob(j.ID)
	find, err := store.Get(k, &h)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", k, err)
	}
	if find {
		j.BookedBy = sdk.BookedBy{
			Name: h.Name,
			ID:   h.ID,
		}
	}
}

// replaceWorkflowJobRunInQueue restart workflow node job
func replaceWorkflowJobRunInQueue(db gorp.SqlExecutor, wNodeJob sdk.WorkflowNodeJobRun) error {
	query := "UPDATE workflow_node_run_job SET status = $1, retry = $2, worker_id = NULL WHERE id = $3"
	if _, err := db.Exec(query, sdk.StatusWaiting, wNodeJob.Retry+1, wNodeJob.ID); err != nil {
		return sdk.WrapError(err, "Unable to set workflow_node_run_job id %d with status %s", wNodeJob.ID, sdk.StatusWaiting)
	}

	query = "UPDATE worker SET status = $2, job_run_id = NULL where job_run_id = $1"
	if _, err := db.Exec(query, wNodeJob.ID, sdk.StatusDisabled); err != nil {
		return sdk.WrapError(err, "Unable to set workers")
	}

	return nil
}
