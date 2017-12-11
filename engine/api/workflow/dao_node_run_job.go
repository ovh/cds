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
	"github.com/ovh/cds/sdk"
)

const loadNodeJobRun = `from workflow_node_run_job
join workflow_node_run on workflow_node_run.id = workflow_node_run_job.workflow_node_run_id
join workflow_run on workflow_run.id = workflow_node_run.workflow_run_id
join workflow on workflow.id = workflow_run.workflow_id
join project on project.id = workflow.project_id
join project_group on project_group.project_id = project.id
where (
	project_group.group_id = ANY(string_to_array($1, ',')::int[])
	or
	true = $4
)
and workflow_node_run_job.queued >= $2
and workflow_node_run_job.status = ANY(string_to_array($3, ','))`

// loadPrepareGroup returns true if groupsID contains shareInfraGroup
// and list of groups, comma separated
func loadPrepareGroup(groupsID []int64) (bool, string) {
	var groupsIDString string
	var isSharedInfraGroup bool
	for i, g := range groupsID {
		if i == 0 {
			groupsIDString = fmt.Sprintf("%d", g)
		} else {
			groupsIDString += "," + fmt.Sprintf("%d", g)
		}
		if g == group.SharedInfraGroup.ID {
			isSharedInfraGroup = true
			break
		}
	}
	return isSharedInfraGroup, groupsIDString
}

// CountNodeJobRunQueue count all workflow_node_run_job accessible
func CountNodeJobRunQueue(db gorp.SqlExecutor, store cache.Store, groupsID []int64, since *time.Time, statuses ...string) (sdk.WorkflowNodeJobRunCount, error) {
	if since == nil {
		since = new(time.Time)
	}

	if len(statuses) == 0 {
		statuses = []string{sdk.StatusWaiting.String()}
	}

	query := "select count(1) " + loadNodeJobRun
	c := sdk.WorkflowNodeJobRunCount{}
	isSharedInfraGroup, groupsIDString := loadPrepareGroup(groupsID)
	count, err := db.SelectInt(query, groupsIDString, since, strings.Join(statuses, ","), isSharedInfraGroup)
	if err != nil {
		return c, sdk.WrapError(err, "workflow.LoadNodeJobRun> Unable to load job runs (Select)")
	}
	c.Count = count
	c.Since = *since
	return c, nil
}

// LoadNodeJobRunQueue load all workflow_node_run_job accessible
func LoadNodeJobRunQueue(db gorp.SqlExecutor, store cache.Store, groupsID []int64, since *time.Time, statuses ...string) ([]sdk.WorkflowNodeJobRun, error) {
	if since == nil {
		since = new(time.Time)
	}

	if len(statuses) == 0 {
		statuses = []string{sdk.StatusWaiting.String()}
	}

	query := "select distinct workflow_node_run_job.* " + loadNodeJobRun
	isSharedInfraGroup, groupsIDString := loadPrepareGroup(groupsID)

	sqlJobs := []JobRun{}
	if _, err := db.Select(&sqlJobs, query, groupsIDString, *since, strings.Join(statuses, ","), isSharedInfraGroup); err != nil {
		return nil, sdk.WrapError(err, "workflow.LoadNodeJobRun> Unable to load job runs (Select)")
	}

	jobs := make([]sdk.WorkflowNodeJobRun, len(sqlJobs))
	for i := range sqlJobs {
		getHatcheryInfo(store, &sqlJobs[i])
		if err := sqlJobs[i].PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "workflow.LoadNodeJobRun> Unable to load job runs (PostGet)")
		}
		jobs[i] = sdk.WorkflowNodeJobRun(sqlJobs[i])
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

	query := "update workflow_node_run_job set job = $2, variables = $3 where id = $1"
	if n, err := s.Exec(query, j.ID, jobJSON, paramsJSON); err != nil {
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
	query := "SELECT job, variables FROM workflow_node_run_job WHERE id = $1"
	var params, job []byte
	if err := s.QueryRow(query, j.ID).Scan(&job, &params); err != nil {
		return sdk.WrapError(err, "PostGet> s.QueryRow id:%d", j.ID)
	}
	if err := json.Unmarshal(job, &j.Job); err != nil {
		return sdk.WrapError(err, "PostGet> json.Unmarshal job")
	}
	if err := json.Unmarshal(params, &j.Parameters); err != nil {
		return sdk.WrapError(err, "PostGet> json.Unmarshal params")
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
